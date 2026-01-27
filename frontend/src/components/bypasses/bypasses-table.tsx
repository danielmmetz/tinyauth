import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { toast } from "sonner";
import { Trash2, Plus } from "lucide-react";

import type { Bypass } from "@/types/bypass";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const calculateExpiresAt = (duration: string): number => {
  const now = Math.floor(Date.now() / 1000);
  const durations: Record<string, number> = {
    "1h": 3600,
    "1d": 86400,
    "1w": 604800,
    "1m": 2592000,
  };
  return now + (durations[duration] || 3600);
};

export const BypassesTable: React.FC = () => {
  const queryClient = useQueryClient();
  const [isSubmittingForm, setIsSubmittingForm] = useState(false);
  const [formData, setFormData] = useState({
    cidr: "",
    domain: "*",
    duration: "1h",
    note: "",
    createdBy: "",
  });

  // Fetch bypasses
  const {
    data: response,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["bypasses"],
    queryFn: () =>
      axios.get("/api/bypasses").then(
        (res) =>
          res.data as {
            bypasses: Bypass[];
            isAdmin: boolean;
            clientIP: string;
          },
      ),
  });

  const bypasses = response?.bypasses || [];
  const isAdmin = response?.isAdmin || false;
  const clientIP = response?.clientIP || "";

  // Fetch domains
  const { data: domainsData } = useQuery({
    queryKey: ["bypass-domains"],
    queryFn: () =>
      axios.get("/api/bypasses/domains").then((res) => res.data.domains || []),
  });

  // Auto-populate CIDR for non-admins
  const displayCidr = !isAdmin ? clientIP : formData.cidr;

  // Delete bypass mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      return axios.delete(`/api/bypasses/${id}`);
    },
    onSuccess: () => {
      toast.success("Bypass deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["bypasses"] });
    },
    onError: (error) => {
      const message =
        axios.isAxiosError(error) && error.response?.data?.message
          ? error.response.data.message
          : "Failed to delete bypass";
      toast.error(message);
    },
  });

  // Create bypass mutation
  const createMutation = useMutation({
    mutationFn: async () => {
      const expiresAt = calculateExpiresAt(formData.duration);
      return axios.post("/api/bypasses", {
        cidr: displayCidr,
        domain: formData.domain,
        expiresAt,
        note: formData.note,
        createdBy: formData.createdBy,
      });
    },
    onSuccess: () => {
      toast.success("Bypass created successfully");
      setFormData({
        cidr: "",
        domain: "*",
        duration: "1h",
        note: "",
        createdBy: "",
      });
      queryClient.invalidateQueries({ queryKey: ["bypasses"] });
    },
    onError: (error) => {
      const message =
        axios.isAxiosError(error) && error.response?.data?.message
          ? error.response.data.message
          : "Failed to create bypass";
      toast.error(message);
    },
  });

  const handleDelete = (id: number) => {
    if (confirm("Are you sure you want to delete this bypass?")) {
      deleteMutation.mutate(id);
    }
  };

  const handleSubmitForm = async () => {
    if (!formData.domain) {
      toast.error("Domain is required");
      return;
    }

    setIsSubmittingForm(true);
    try {
      await createMutation.mutateAsync();
    } finally {
      setIsSubmittingForm(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <p className="text-muted-foreground">Loading bypasses...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-8">
        <p className="text-destructive">Failed to load bypasses</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto border rounded-lg">
      <table className="w-full text-sm">
        <thead className="bg-muted border-b">
          <tr>
            <th className="px-4 py-3 text-left font-medium">CIDR / IP</th>
            <th className="px-4 py-3 text-left font-medium">Domain</th>
            <th className="px-4 py-3 text-left font-medium">Expires At</th>
            <th className="px-4 py-3 text-left font-medium">Note</th>
            {isAdmin && (
              <th className="px-4 py-3 text-left font-medium">Created By</th>
            )}
            <th className="px-4 py-3 text-left font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {bypasses.length === 0 ? (
            <tr>
              <td colSpan={isAdmin ? 6 : 5} className="px-4 py-8 text-center">
                <p className="text-muted-foreground">
                  No IP bypasses created yet
                </p>
              </td>
            </tr>
          ) : (
            bypasses.map((bypass) => {
              const expiresAt = new Date(bypass.expiresAt * 1000);
              const isExpired = expiresAt < new Date();

              return (
                <tr key={bypass.id} className="border-b hover:bg-muted/50">
                  <td className="px-4 py-3 font-mono text-xs">{bypass.cidr}</td>
                  <td className="px-4 py-3">{bypass.domain}</td>
                  <td className="px-4 py-3">
                    <span className={isExpired ? "text-destructive" : ""}>
                      {expiresAt.toLocaleString()}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {bypass.note || "-"}
                  </td>
                  {isAdmin && (
                    <td className="px-4 py-3 text-muted-foreground">
                      {bypass.createdBy}
                    </td>
                  )}
                  <td className="px-4 py-3">
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleDelete(bypass.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              );
            })
          )}

          {/* Form row for creating new bypass */}
          <tr className="border-b bg-muted/30 hover:bg-muted/50">
            <td className="px-4 py-3">
              <Input
                placeholder={clientIP}
                disabled={!isAdmin}
                value={displayCidr}
                onChange={(e) => {
                  if (isAdmin) {
                    setFormData((prev) => ({ ...prev, cidr: e.target.value }));
                  }
                }}
                className="h-9"
              />
            </td>
            <td className="px-4 py-3">
              <Select
                value={formData.domain}
                onValueChange={(value) => {
                  setFormData((prev) => ({ ...prev, domain: value }));
                }}
              >
                <SelectTrigger className="h-9">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {domainsData?.map((domain: string) => (
                    <SelectItem key={domain} value={domain}>
                      {domain}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </td>
            <td className="px-4 py-3">
              <Select
                value={formData.duration}
                onValueChange={(value) => {
                  setFormData((prev) => ({ ...prev, duration: value }));
                }}
              >
                <SelectTrigger className="h-9">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1h">1 hour</SelectItem>
                  <SelectItem value="1d">1 day</SelectItem>
                  <SelectItem value="1w">1 week</SelectItem>
                  <SelectItem value="1m">1 month</SelectItem>
                </SelectContent>
              </Select>
            </td>
            <td className="px-4 py-3">
              <Input
                placeholder="Optional note"
                value={formData.note}
                onChange={(e) => {
                  setFormData((prev) => ({ ...prev, note: e.target.value }));
                }}
                className="h-9"
              />
            </td>
            {isAdmin && (
              <td className="px-4 py-3">
                <Input
                  placeholder="Created by (optional)"
                  value={formData.createdBy}
                  onChange={(e) => {
                    setFormData((prev) => ({
                      ...prev,
                      createdBy: e.target.value,
                    }));
                  }}
                  className="h-9"
                />
              </td>
            )}
            <td className="px-4 py-3">
              <Button
                size="sm"
                onClick={handleSubmitForm}
                disabled={isSubmittingForm || createMutation.isPending}
              >
                <Plus className="h-4 w-4" />
              </Button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  );
};
