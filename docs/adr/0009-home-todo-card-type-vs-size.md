# Home todo cards: server sends type + order; client owns size and packing

Homepage actionable cards (新伙伴待确认, etc.) need small/medium/large footprints in a 2×2 cell page with horizontal paging. We decided the BFF returns an ordered list of cards keyed by **type** (plus copy, count, `action_route`), sorted large-tier types before medium before small, and **does not** send `size` or pre-built `pages[]`. The client maps type → size (大=4 cells, 中=2/整行, 小=1) and packs pages locally.

Rejected: server-authored `size` (duplicates the type→size table across clients and invites drift), and server-prepacked `pages[]` (locks the API to a 2×2 grid and forces a version bump for any layout tweak). Flutter and KMP share the same packing rules from the type table instead.
