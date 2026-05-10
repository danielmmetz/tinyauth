type IuseRedirectUri = {
  url?: URL;
  valid: boolean;
  trusted: boolean;
  allowedProto: boolean;
  httpsDowngrade: boolean;
};

export const useRedirectUri = (
  redirect_uri: string | null,
  cookieDomain: string,
): IuseRedirectUri => {
  let isValid = false;
  let isTrusted = false;
  let isAllowedProto = false;
  let isHttpsDowngrade = false;

  if (!redirect_uri) {
    return {
      valid: isValid,
      trusted: isTrusted,
      allowedProto: isAllowedProto,
      httpsDowngrade: isHttpsDowngrade,
    };
  }

  let url: URL;

  // Support relative paths (e.g. "/bypasses") by resolving them against the
  // current origin. They cannot redirect off-origin so they're trusted.
  const isRelativePath = redirect_uri.startsWith("/");

  try {
    url = isRelativePath
      ? new URL(redirect_uri, window.location.origin)
      : new URL(redirect_uri);
  } catch {
    return {
      valid: isValid,
      trusted: isTrusted,
      allowedProto: isAllowedProto,
      httpsDowngrade: isHttpsDowngrade,
    };
  }

  isValid = true;

  if (
    isRelativePath ||
    url.hostname == cookieDomain ||
    url.hostname.endsWith(`.${cookieDomain}`)
  ) {
    isTrusted = true;
  }

  if (url.protocol == "http:" || url.protocol == "https:") {
    isAllowedProto = true;
  }

  if (window.location.protocol == "https:" && url.protocol == "http:") {
    isHttpsDowngrade = true;
  }

  return {
    url,
    valid: isValid,
    trusted: isTrusted,
    allowedProto: isAllowedProto,
    httpsDowngrade: isHttpsDowngrade,
  };
};
