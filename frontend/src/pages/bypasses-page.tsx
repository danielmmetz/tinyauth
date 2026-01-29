import { Navigate, useLocation } from "react-router";
import { useUserContext } from "@/context/user-context";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { BypassesTable } from "@/components/bypasses/bypasses-table";

export const BypassesPage = () => {
  const { isLoggedIn } = useUserContext();
  const location = useLocation();

  if (!isLoggedIn) {
    const returnUrl = `${location.pathname}${location.search}`;
    return <Navigate to={`/login?redirect_uri=${encodeURIComponent(returnUrl)}`} replace />;
  }

  return (
    <Card className="w-full max-w-6xl">
      <CardHeader>
        <CardTitle className="text-3xl">IP Bypasses</CardTitle>
        <CardDescription>
          Manage temporary IP bypasses for authentication
        </CardDescription>
      </CardHeader>
      <CardContent>
        <BypassesTable />
      </CardContent>
    </Card>
  );
};
