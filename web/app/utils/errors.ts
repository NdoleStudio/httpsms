import Bag from "~/utils/bag";
import { capitalize } from "~/utils/capitalize";

export class ErrorMessages extends Bag<string> {}

const sanitize = (key: string, values: Array<string>): Array<string> => {
  return values.map((value: string) => {
    return capitalize(
      value
        .split(key)
        .join(key.replace("_", " "))
        .split("_")
        .join(" ")
        .split("-")
        .join(" ")
        .split(" char")
        .join(" character")
        .split(" field ")
        .join(" "),
    );
  });
};

interface AxiosLikeError {
  response?: {
    data?: { data?: Record<string, string[]> };
    status?: number;
  };
}

export const getErrorMessages = (error: AxiosLikeError): ErrorMessages => {
  const errors = new ErrorMessages();
  if (
    error === null ||
    typeof error.response?.data?.data !== "object" ||
    error.response?.data?.data === null ||
    error.response?.status !== 422
  ) {
    return errors;
  }

  Object.keys(error.response.data.data).forEach((key: string) => {
    errors.addMany(key, sanitize(key, error.response!.data!.data![key]));
  });

  return errors;
};
