+++
type = "added"
public_api = "add"
behavior = "new"
+++

Derive release impact from semantic fragment levers instead of storing an explicit bump in each fragment.

Fragments can now describe `public_api`, `behavior`, `dependency`, and `runtime`, and `changes` combines those facts with the repository's public-API stability policy to recommend the next version.
