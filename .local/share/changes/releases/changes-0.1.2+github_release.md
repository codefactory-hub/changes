# Release 0.1.2

## Fixed

- Omit unset integer ordering fields from fragment front matter. `changes create` no longer serializes `release_notes_priority = 0` or `display_order = 0` when those flags were not provided on the command line.
