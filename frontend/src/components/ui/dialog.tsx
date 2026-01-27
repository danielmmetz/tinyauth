import * as React from "react";
import { cn } from "@/lib/utils";

interface DialogProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children: React.ReactNode;
}

interface DialogContentProps {
  children: React.ReactNode;
  className?: string;
}

interface DialogHeaderProps {
  children: React.ReactNode;
  className?: string;
}

interface DialogTitleProps {
  children: React.ReactNode;
  className?: string;
}

interface DialogDescriptionProps {
  children: React.ReactNode;
  className?: string;
}

export const Dialog: React.FC<DialogProps> = ({
  open = false,
  onOpenChange,
  children,
}) => {
  const [isOpen, setIsOpen] = React.useState(open);

  React.useEffect(() => {
    setIsOpen(open);
  }, [open]);

  const handleOpenChange = (newOpen: boolean) => {
    setIsOpen(newOpen);
    onOpenChange?.(newOpen);
  };

  const providerValue = React.useMemo(
    () => ({ isOpen, onOpenChange: handleOpenChange }),
    [isOpen]
  );

  return (
    <DialogProvider value={providerValue}>{children}</DialogProvider>
  );
};

interface DialogContextType {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

const DialogContext = React.createContext<DialogContextType | undefined>(
  undefined
);

const DialogProvider: React.FC<{
  value: DialogContextType;
  children: React.ReactNode;
}> = ({ value, children }) => (
  <DialogContext.Provider value={value}>{children}</DialogContext.Provider>
);

const useDialog = () => {
  const context = React.useContext(DialogContext);
  if (!context) {
    throw new Error("useDialog must be used within a Dialog");
  }
  return context;
};

export const DialogTrigger: React.FC<
  React.ButtonHTMLAttributes<HTMLButtonElement>
> = ({ children, onClick, ...props }) => {
  const { onOpenChange } = useDialog();

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    onClick?.(e);
    onOpenChange(true);
  };

  return (
    <button onClick={handleClick} {...props}>
      {children}
    </button>
  );
};

export const DialogContent: React.FC<DialogContentProps> = ({
  children,
  className,
}) => {
  const { isOpen, onOpenChange } = useDialog();

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onOpenChange(false);
      }
    };

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      document.body.style.overflow = "hidden";
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "auto";
    };
  }, [isOpen, onOpenChange]);

  if (!isOpen) {
    return null;
  }

  return (
    <>
      <div
        className="fixed inset-0 z-40 bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      <div
        className={cn(
          "fixed left-1/2 top-1/2 z-50 w-full max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-6 shadow-lg duration-200 ease-out data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]",
          className
        )}
      >
        {children}
      </div>
    </>
  );
};

export const DialogHeader: React.FC<DialogHeaderProps> = ({
  children,
  className,
}) => (
  <div className={cn("flex flex-col gap-2", className)}>{children}</div>
);

export const DialogTitle: React.FC<DialogTitleProps> = ({
  children,
  className,
}) => (
  <h2 className={cn("text-lg font-semibold leading-none tracking-tight", className)}>
    {children}
  </h2>
);

export const DialogDescription: React.FC<DialogDescriptionProps> = ({
  children,
  className,
}) => (
  <p className={cn("text-sm text-muted-foreground", className)}>
    {children}
  </p>
);
