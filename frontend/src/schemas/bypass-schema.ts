import { z } from "zod";

export const bypassSchema = z.object({
  cidr: z.string().min(1, "CIDR is required"),
  domain: z.string().min(1, "Domain is required"),
  duration: z.enum(["1h", "1d", "1w", "1m"]),
  note: z.string(),
});

export type BypassSchema = z.infer<typeof bypassSchema>;
