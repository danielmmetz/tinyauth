# Dynamic IP Bypasses

Dynamic IP bypasses let authenticated users create temporary, time-limited rules that allow specific IPs to skip the login screen. This is useful for apps accessed by clients that can't handle an OAuth flow — IoT devices, API consumers, embedded clients, etc.

Bypasses are scoped to a domain (or `*` for all domains) and expire automatically. The UI offers preset durations of 1 hour, 1 day, 1 week, or 1 month.

## Enabling

The feature is disabled by default. To enable it, set the environment variable:

```
ENABLE_DYNAMIC_IP_BYPASS=true
```

Or pass the `--enable-dynamic-ip-bypass` flag.

## Usage

Once enabled, authenticated users can manage bypasses from the `/bypasses` page in the Tinyauth UI. From there you can create new bypasses, see active ones, and delete them.

## Permissions

There are two levels of access: **admin** and **regular user**.

**Admins** are OAuth users whose provider returns `tinyauth-admin` in their groups claim. Admins can:

- Create bypasses for arbitrary CIDRs (not just their own IP)
- Create bypasses for any domain
- View and delete any user's bypasses

> **Note:** The `tinyauth-admin` group name is currently hard-coded. Making it configurable is a potential future improvement.

**Regular users** can:

- Create bypasses for their own IP address only
- Create bypasses only for domains they've been granted access to
- View and delete only their own bypasses

### Domain allow-lists for non-admin users

Non-admin users are restricted to a set of allowed domains when creating bypasses. This list is read from a `bypass_domains_allowed` claim in the OAuth userinfo response (a comma-separated string of domains). Configuring your OAuth provider to return this claim is provider-specific.
