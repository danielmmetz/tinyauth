export interface Bypass {
  id: number;
  cidr: string;
  domain: string;
  expiresAt: number;
  note: string;
  createdBy: string;
  createdAt: number;
}

export interface BypassesResponse {
  bypasses: Bypass[];
  isAdmin: boolean;
  clientIP: string;
}
