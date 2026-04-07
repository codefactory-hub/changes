+++
public_api = "add"
behavior = "new"
+++

Add flexible global and repo-local layout support for `xdg` and `home`.

`changes` can now resolve configuration, data, and state through either XDG-style directories or a single-root home layout, inspect authority with `changes doctor`, and fail loudly when multiple supported layouts compete.
