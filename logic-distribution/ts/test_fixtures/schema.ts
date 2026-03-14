/** Test: Schema/Validation → SCHEMA_VALIDATION */
import { z } from "zod";

export const userSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
  age: z.number().int().positive(),
});

export function validateUser(data: unknown) {
  return userSchema.parse(data);
}

export function safeValidateUser(data: unknown) {
  return userSchema.safeParse(data);
}
