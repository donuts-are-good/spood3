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

## Incentives

- Hybrid stats average parent's stats and gain a +10 infusion randomly distributed across stats, making them slightly stronger than baseline custom created fighters.  

## Open Questions

- None; current concerns are parked until real-world usage requires adjustments.

## Next Steps

1. Design sponsorship DB schema & API.  
2. Implement hybrid wizard (backend + frontend).  
3. Update templates to show lineage badges.  
4. Draft lore blog post once mechanics land.

