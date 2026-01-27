import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { toast } from "sonner";

import { bypassSchema, type BypassSchema } from "@/schemas/bypass-schema";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";

interface BypassDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

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

export const BypassDialog: React.FC<BypassDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<BypassSchema>({
    resolver: zodResolver(bypassSchema),
    defaultValues: {
      cidr: "",
      domain: "*",
      duration: "1h",
      note: "",
    },
  });

  // Fetch domains
  const { data: domainsData } = useQuery({
    queryKey: ["bypass-domains"],
    queryFn: () =>
      axios.get("/api/bypasses/domains").then((res) => res.data.domains || []),
  });

  // Fetch bypasses to get isAdmin flag and clientIP
  const { data: bypassesData } = useQuery({
    queryKey: ["bypasses"],
    queryFn: () =>
      axios.get("/api/bypasses").then((res) => res.data as {
        isAdmin: boolean;
        clientIP: string;
      }),
  });

  const isAdmin = bypassesData?.isAdmin || false;
  const clientIP = bypassesData?.clientIP || "";

  // Auto-populate CIDR field for non-admins
  useEffect(() => {
    if (!isAdmin && clientIP && open) {
      form.setValue("cidr", clientIP);
    }
  }, [isAdmin, clientIP, open, form]);

  // Create bypass mutation
  const createMutation = useMutation({
    mutationFn: async (data: BypassSchema) => {
      const expiresAt = calculateExpiresAt(data.duration);
      return axios.post("/api/bypasses", {
        cidr: data.cidr,
        domain: data.domain,
        expiresAt,
        note: data.note,
      });
    },
    onSuccess: () => {
      toast.success("Bypass created successfully");
      form.reset();
      onOpenChange(false);
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

  const onSubmit = async (data: BypassSchema) => {
    setIsSubmitting(true);
    try {
      await createMutation.mutateAsync(data);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create IP Bypass</DialogTitle>
          <DialogDescription>
            Create a temporary IP bypass to allow access without authentication
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {/* CIDR Field */}
            <FormField
              control={form.control}
              name="cidr"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>CIDR / IP Address</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="192.168.1.0/24 or 10.0.0.1"
                      disabled={!isAdmin}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Domain Field */}
            <FormField
              control={form.control}
              name="domain"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Domain</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a domain" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {domainsData?.map((domain: string) => (
                        <SelectItem key={domain} value={domain}>
                          {domain}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Duration Field */}
            <FormField
              control={form.control}
              name="duration"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Duration</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select duration" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="1h">1 hour</SelectItem>
                      <SelectItem value="1d">1 day</SelectItem>
                      <SelectItem value="1w">1 week</SelectItem>
                      <SelectItem value="1m">1 month</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Note Field */}
            <FormField
              control={form.control}
              name="note"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Note (optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="Add a note for this bypass" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Submit Button */}
            <Button
              type="submit"
              disabled={isSubmitting || createMutation.isPending}
              className="w-full"
            >
              {isSubmitting || createMutation.isPending
                ? "Creating..."
                : "Create Bypass"}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
