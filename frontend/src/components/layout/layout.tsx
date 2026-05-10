import { useAppContext } from "@/context/app-context";
import { LanguageSelector } from "../language/language";
import { Outlet, useLocation } from "react-router";
import { useCallback, useEffect, useState } from "react";
import { DomainWarning } from "../domain-warning/domain-warning";
import { ThemeToggle } from "../theme-toggle/theme-toggle";

// Routes that need a wider container than the standard login-form width.
const WIDE_ROUTES = ["/bypasses"];

const BaseLayout = ({ children }: { children: React.ReactNode }) => {
  const { backgroundImage, title } = useAppContext();
  const { pathname } = useLocation();

  useEffect(() => {
    document.title = title;
  }, [title]);

  const isWideRoute = WIDE_ROUTES.some((route) => pathname.startsWith(route));
  const containerClass = isWideRoute
    ? "w-full max-w-6xl my-8"
    : "max-w-sm md:min-w-sm min-w-xs";

  return (
    <div
      className="flex flex-col justify-center items-center min-h-svh px-4"
      style={{
        backgroundImage: `url(${backgroundImage})`,
        backgroundSize: "cover",
        backgroundPosition: "center",
      }}
    >
      <div className="absolute top-4 right-4 flex flex-row gap-2">
        <ThemeToggle />
        <LanguageSelector />
      </div>
      <div className={containerClass}>{children}</div>
    </div>
  );
};

export const Layout = () => {
  const { appUrl, warningsEnabled } = useAppContext();
  const [ignoreDomainWarning, setIgnoreDomainWarning] = useState(() => {
    return window.sessionStorage.getItem("ignoreDomainWarning") === "true";
  });
  const currentUrl = window.location.origin;

  const handleIgnore = useCallback(() => {
    window.sessionStorage.setItem("ignoreDomainWarning", "true");
    setIgnoreDomainWarning(true);
  }, [setIgnoreDomainWarning]);

  if (!ignoreDomainWarning && warningsEnabled && appUrl !== currentUrl) {
    return (
      <BaseLayout>
        <DomainWarning
          appUrl={appUrl}
          currentUrl={currentUrl}
          onClick={() => handleIgnore()}
        />
      </BaseLayout>
    );
  }

  return (
    <BaseLayout>
      <Outlet />
    </BaseLayout>
  );
};
