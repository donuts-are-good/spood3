# Hybrid Fighter Program

## Lore / Fantasy

- **Research Permit**  
  - Wealthy degenerates fund their favorite combatant by purchasing a research permit in the Chaos Marketplace.  
  - Holding a permit grants “bio-rights” to that fighter’s genome. Think of it as notarized permission to meddle with their DNA.  
- **Rogue Lab**  
  - Once a patron holds *two or more* permits, a shady biotech crew offers to blend their “licensed genomes.”  
  - The Department calls this “Familial Harmonization”; the underbelly calls it “Hybrid Theory.”  
  - Hybrids inherit a lineage badge listing all parent permits involved. Future lore hook: hybrids might unlock hidden traits when multiple generations stack.

## Proposed Shop Items

| Item | Type | Cost | Notes |
| --- | --- | --- | --- |
| **Research Permit** | `fighter_sponsorship` | 200K credits | *Genetic Research Licence* Item limit: one in inventory at a time; cannot purchase another until the current one is used. Consumed when you assign it, permanently recording license for that fighter. |
| **Rogue Lab** | `genetic_splicer` | 500K credits | *Disposable lab kit* Hidden “back-alley” shop section; item only visible after owning ≥1 licensed fighter. Consumed per hybrid; requires ≥2 licensed fighters in the user’s dossier. No throttles initially—monitor for spam before adding caps. |

## Mechanics

### Shop Presentation
1. **Primary shop grid** always shows Research Permit for authenticated users who do not already hold an unused permit, just like the combat license lvl 1, which appears inactive if it is already in the inventory. 
2. Users are limited to one unassigned permit at a time (like Combat License).  
3. When a user purchases their first permit, the page reveals a secondary “back-alley” shop section (purple styling, trunk-in-the-alley vibe) directly below the main grid.  
4. The Rogue Lab product tile lives exclusively in this secondary section and only renders if the user has at least one fighter already licensed via Research Permit.  
5. The reveal happens without a page reload. front-end listens for purchase success and toggles the hidden section.  
6. Users without an active Research Permit (including spectators / logged-out visitors) never see the back-alley section rendered.

### Sponsorship Flow
1. Buy permit → inventory card (similar to Combat License).  
2. Permits sit in a pending state; while one is unused the shop disables purchasing another.  
3. Clicking opens `/user/sponsorships` wizard.  
4. User selects any active fighter, must be status ACTIVE, no DEAD, DECEASED, or UNDEAD.  
5. Repository stores table `sponsorships(user_id, fighter_id, created_at)` with a unique `(user_id, fighter_id)` constraint.  
6. Consuming the permit removes it from inventory; licensing lives as durable, non-transferable data tied to that fighter/user forever.  
7. UI displays licensed fighters & lineage rights.  
8. All licenses and inventory items are non-transferable; only the purchasing user can use them.

### Hybrid Creation Flow
1. Preconditions  
   - User owns ≥2 sponsorship entries (licenses must belong to the same user; no borrowing).  
   - User owns ≥1 Rogue Lab item.  
2. User enters hybrid wizard, chooses two sponsored fighters that the user has sponsored (A+B).  
3. System pulls both fighters’ combat stats and chaos metadata.  
4. **Stat math:**  
   - Compute `avgStrength = round((A.Strength + B.Strength) / 2)` (repeat for speed/endurance/technique).  
   - Compute `baseTotal = sum(avg stats)`; compute `parentTotal = average(total stats of A and B)`; difference typically small.  
   - Add **bonus +10 points** distributed randomly (one point at a time) across the four stats.  
   - Do not clamp each stat 20–130 to avoid extremes, let it get wild.
5. Chaos stats: either randomly roll brand-new values or derive fun blends (e.g., blood-type mashup). TBD but note lineage references.  
6. Persist new fighter with `Ancestor1ID`, `Ancestor2ID`, `CreatedByUserID`.  
7. Consume one Rogue Lab item. Permits remain as historical licenses for each fighter.

### Lineage Display
- Extend fighter metadata with ancestor IDs and the username of the owner of the permits used.  
- Templates show “Genome licensed to @username for Fighter A + Fighter B” (single owner because both permits must be in one inventory).  
- Maintain ancestry list to show hybrid trees on fighter pages for genetic ancestors and hybrid offspring. Use a directed acylic graph family tree or sankey.

## Database Schema

1. **sponsorships**  
   - Go struct:
     ```go
     type Sponsorship struct {
         ID        int       `db:"id"`
         UserID    int       `db:"user_id"`
         FighterID int       `db:"fighter_id"`
         CreatedAt time.Time `db:"created_at"`
     }
     ```
   - Unique `(user_id, fighter_id)` constraint prevents duplicate licenses.
   - Fighters must be `status = ACTIVE` at time of insertion; no undead or dead ids allowed.
   - Drives “licensed fighters” list in the UI; deleting rows is verboten unless fighter is purged.

2. **fighters table augmentation**  
   - Add lineage columns directly to existing `fighters` table to keep parity with current model. No NULLs—use sentinel defaults so templates can rely on zero checks:
     - `ancestor1_id` (int, default **0** meaning “no ancestor”)  
     - `ancestor2_id` (int, default **0**)  
     - `hybrid_created_by_user_id` (int, default **0**)  
     - `hybrid_rogue_lab_inventory_id` (int, default **0**)  
   - Corresponding Go struct additions:
     ```go
     type Fighter struct {
         // existing fields...
         Ancestor1ID               int       `db:"ancestor1_id"`
         Ancestor2ID               int       `db:"ancestor2_id"`
         HybridCreatedByUserID     int       `db:"hybrid_created_by_user_id"`
         HybridRogueLabInventoryID int       `db:"hybrid_rogue_lab_inventory_id"`
     }
     ```
   - Hybrid creation time piggybacks on the existing `fighters.created_at` field, so no extra timestamp column is required.
   - Backfill historical fighters with the sentinel defaults (0) so hybrid detection logic is a simple `if fighter.Ancestor1ID == 0` check.
   - This keeps hybrids as first-class fighters, avoids extra join tables, and matches prior patterns (e.g., `CreatedByUserID` already lives on fighters).

3. **fighter_lineage_view** (optional materialized view)  
   - Updated view would simply denormalize the new fighter columns to fetch ancestor names/usernames quickly for UI graphs.

Existing tables continue to handle inventory: purchasing a Research Permit inserts into `user_inventory`; assigning a fighter deletes that inventory row and inserts into `sponsorships`. Rogue Labs live in `user_inventory` until consumed; upon hybrid creation, decrement quantity and log the `user_inventory.id` in `fighter_hybrids.rogue_lab_inventory_id`.

### Schema Parity Check

**Similarities with existing patterns**
- Mirrors `user_inventory` consumption flow already used for Combat Licenses and serums (pending inventory row consumed by dedicated table write).
- Uses familiar `(user_id, fighter_id)` uniqueness constraint like betting or MVP records to keep one-row-per-relationship semantics.
- `fighter_hybrids` structure parallels `fighter_kills`/`champion_legacy` tables by storing FK references plus metadata (`created_by_user_id`, timestamps) for audit trails.
- API request/response shapes follow existing JSON endpoints (`/fight/apply-effect`, shop purchase) that validate ownership, check inventory, then mutate DB inside a transaction.

**Intentional differences**
- Introduces explicit lineage tracking (`ancestor1_id`, `ancestor2_id`) which is new versus other features; needed to render genealogy graphs and prevent re-parenting.
- Links Rogue Lab consumption back to the specific `user_inventory` row for compliance/auditing—other items typically just decrement quantity without logging the inventory row id.
- Adds read-only lineage endpoint exposed publicly; most current user tools are private. This one leaks structured ancestry data to front-end widgets.
- Research Permit licensing requires fighter status validation (ACTIVE only) which is stricter than other items (e.g., blessings/curses can target undead). This ensures lore consistency.
- Fighters use sentinel defaults (zero IDs) instead of NULLs, same as other tables—no lingering NullTypes in structs.

## API Surface

- `GET /user/licenses` — returns pending permits, count of licensed fighters, and roster of eligible fighters (active only). Powers the wizard list.
- `POST /user/licenses` — body `{ "fighter_id": <int> }`. Validates ownership of an unused permit, fighter eligibility, and uniqueness, then consumes the permit and creates the sponsorship row.
- `GET /user/hybrids/options` — returns `{ licensed_fighters: [...], rogue_labs_available: <int> }` for the hybrid modal. Used to gate the “mix” button.
- `POST /user/hybrids` — body `{ "ancestor1_id": <int>, "ancestor2_id": <int> }`. Checks both IDs belong to the requesting user, ensures at least one Rogue Lab in inventory, runs the stat-mixing routine, creates the fighter + `fighter_hybrids` entry, and decrements inventory. Response returns new fighter payload for redirect.
- `GET /fighter/{id}/lineage` (public) — JSON describing ancestors/descendants for the fighter page widget; pulls from the view/table above.

All endpoints live behind authentication (except the read-only lineage endpoint) and reuse existing middleware so that only the owner can create licenses or hybrids.

## Incentives

- Hybrid stats average parent's stats and gain a +10 infusion randomly distributed across stats, making them slightly stronger than baseline custom created fighters.  

## Open Questions

- None; current concerns are parked until real-world usage requires adjustments.

## Next Steps

1. Design sponsorship DB schema & API.  
2. Implement hybrid wizard (backend + frontend).  
3. Update templates to show lineage badges.  
4. Draft lore blog post once mechanics land.

