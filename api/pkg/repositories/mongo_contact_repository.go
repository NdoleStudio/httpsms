package repositories

import (
	"context"
	"fmt"
	"regexp"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mongoContactRepository is responsible for persisting entities.Contact in MongoDB.
type mongoContactRepository struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	collection *mongo.Collection
}

// NewMongoContactRepository creates the MongoDB version of the ContactRepository.
func NewMongoContactRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *mongo.Database,
) ContactRepository {
	return &mongoContactRepository{
		logger:     logger.WithService(fmt.Sprintf("%T", &mongoContactRepository{})),
		tracer:     tracer,
		collection: db.Collection(collectionContacts),
	}
}

func (repository *mongoContactRepository) Store(ctx context.Context, contacts []*entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if len(contacts) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	documents := make([]any, len(contacts))
	for index, contact := range contacts {
		documents[index] = contact
	}

	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts", len(contacts)))
	}

	return nil
}

func (repository *mongoContactRepository) Update(ctx context.Context, contact *entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := mongoContactIDFilter(contact.UserID, contact.ID)
	if _, err := repository.collection.ReplaceOne(ctx, filter, contact); err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact with ID [%s]", contact.ID))
	}

	return nil
}

func (repository *mongoContactRepository) Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	contact := new(entities.Contact)
	err := repository.collection.FindOne(ctx, mongoContactIDFilter(userID, contactID)).Decode(contact)
	if err == mongo.ErrNoDocuments {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "contact with ID [%s] for user [%s] does not exist", contactID, userID))
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load contact with ID [%s] for user [%s]", contactID, userID))
	}

	return contact, nil
}

func (repository *mongoContactRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	findOptions := options.Find().
		SetSort(mongoContactSort(params)).
		SetSkip(int64(params.Skip)).
		SetLimit(int64(params.Limit))
	cursor, err := repository.collection.Find(ctx, mongoContactFilter(userID, params.Query), findOptions)
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot index contacts for user [%s] with params [%+#v]", userID, params))
	}
	defer cursor.Close(ctx)

	contacts := make([]entities.Contact, 0)
	if err = cursor.All(ctx, &contacts); err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode contacts for user [%s]", userID))
	}

	return &contacts, nil
}

func (repository *mongoContactRepository) Count(ctx context.Context, userID entities.UserID, params IndexParams) (int64, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	count, err := repository.collection.CountDocuments(ctx, mongoContactFilter(userID, params.Query))
	if err != nil {
		return 0, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot count contacts for user [%s] with query [%s]", userID, params.Query))
	}

	return count, nil
}

func (repository *mongoContactRepository) FetchByPhoneNumbers(ctx context.Context, userID entities.UserID, phoneNumbers []string) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	findOptions := options.Find().SetSort(bson.D{
		{Key: "updated_at", Value: 1},
		{Key: "_id", Value: 1},
	})
	cursor, err := repository.collection.Find(ctx, mongoContactPhoneNumbersFilter(userID, phoneNumbers), findOptions)
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch contacts for user [%s] by phone numbers [%v]", userID, phoneNumbers))
	}
	defer cursor.Close(ctx)

	contacts := make([]entities.Contact, 0)
	if err = cursor.All(ctx, &contacts); err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode contacts for user [%s] by phone numbers", userID))
	}

	return &contacts, nil
}

func (repository *mongoContactRepository) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	if _, err := repository.collection.DeleteOne(ctx, mongoContactIDFilter(userID, contactID)); err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact with ID [%s] for user [%s]", contactID, userID))
	}

	return nil
}

func (repository *mongoContactRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	if _, err := repository.collection.DeleteMany(ctx, bson.D{{Key: "user_id", Value: string(userID)}}); err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all contacts for user [%s]", userID))
	}

	return nil
}

func mongoContactFilter(userID entities.UserID, query string) bson.D {
	filter := bson.D{{Key: "user_id", Value: string(userID)}}
	if query == "" {
		return filter
	}

	expression := bson.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
	return append(filter, bson.E{Key: "$or", Value: bson.A{
		bson.D{{Key: "name", Value: expression}},
		bson.D{{Key: "emails", Value: expression}},
		bson.D{{Key: "phone_numbers", Value: expression}},
	}})
}

func mongoContactSort(params IndexParams) bson.D {
	sortBy := "updated_at"
	if params.SortBy == "name" {
		sortBy = "name"
	}

	direction := 1
	if params.SortBy == "" || params.SortDescending {
		direction = -1
	}

	return bson.D{
		{Key: sortBy, Value: direction},
		{Key: "_id", Value: direction},
	}
}

func mongoContactPhoneNumbersFilter(userID entities.UserID, phoneNumbers []string) bson.D {
	return bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "phone_numbers", Value: bson.D{{Key: "$in", Value: phoneNumbers}}},
	}
}

func mongoContactIDFilter(userID entities.UserID, contactID uuid.UUID) bson.D {
	return bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "_id", Value: contactID.String()},
	}
}
