export function capitalize(value: string | null): string {
  if (!value) {
    return "";
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export default capitalize;
