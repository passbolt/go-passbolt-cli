// Package util provides shared utilities for the CLI.
//
// # Shared command lifecycle
//
// WithClient(cmd, fn) wraps context creation, login, session-key persistence +
// logout, and SilenceUsage so command RunE functions don't repeat that plumbing.
// Read flags first (so flag errors still print usage), then run the API calls and
// rendering inside fn, which receives the context and a logged-in *api.Client.
//
// # Shared output helpers
//
// Entity commands render results as JSON or pterm tables. These helpers keep that
// output consistent and remove the per-entity duplication; the typed JSON output
// structs (one per entity) stay in their packages and are passed in generically:
//
//   - PrintJSON / FprintJSON marshal a value as indented JSON (single object or
//     array), to Out or an explicit io.Writer.
//   - PrintJSONColumnFiltered keeps only the requested keys when the user passed an
//     explicit --column subset.
//   - PrintTable renders a pterm table from a column list and a per-cell value
//     function; it is generic over the item type.
//   - RenderTable is the single pterm render path, for callers that build their own
//     TableData. It is the only place table cells are sanitized, so callers pass
//     raw values, and it writes to Out.
//   - PermissionsToJSONOutput / PrintPermissionTable render the permission output
//     shared by the resource and folder commands.
//
// # Terminal safety
//
// Passbolt stores names, usernames and URIs verbatim, so anything rendered for a
// human is server-controlled text. Every such write goes through Out or Err, which
// replace control sequences with U+FFFD, and single-line values are narrowed further
// with Sanitize (\n and \t survive the stream writer by design, so a multi-line value
// such as a description is printed without it). Table cells are sanitized once, in
// RenderTable. JSON output escapes those runes instead of replacing them, so the
// document still parses.
//
// Replacing keeps the change visible: "ad\u200bmin" and "admin" used to render
// identically. Execute then flushes and warns, on one line, after the output.
// forbidigo enforces the rule; the exceptions are completions and the exec child's
// stdio, which are machine-facing, and the keepass progressbar, which needs cursor
// control and renders no server text. The SDK's own debug log writes raw server
// responses straight to os.Stdout, so --debug output is not sanitized; it is opt-in
// and for diagnosis, not normal use.
package util
