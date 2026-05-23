# Vessel

A dictation-first thought-collector. The user speaks a thought; the system transcribes it, enriches it (title, summary, category, tags), and — once `done` — generates an **Echo**: a short, constrained AI response intended to make the thought more useful to its author later.

## Language

### Thought

**Thought**:
A single dictated capture. Persisted as audio plus a transcript and one enrichment.
_Avoid_: Note, entry, recording.

**Enrichment**:
The LLM-generated metadata attached to a Thought: title, summary, category, tags.
_Avoid_: Annotation, metadata.

**Category**:
The closed-set classification of a Thought. One of: `idea`, `observation`, `feeling`, `learning`.
_Avoid_: Type, kind.

### Echo

**Echo**:
A short, generated response to a finished Thought. Bounded (≤40 words), English, second person, no advice, no validation phrases, no questions outside Challenger mode. An Echo is supplementary — it never blocks the Thought lifecycle and never appears in the Thought's own status.

**Echo Mode**:
The shape of an Echo. Fixed enum: `mirror`, `challenger`, `reframer`, `extender`. New modes require a schema migration.

**Default Echo**:
The one Echo generated eagerly for every `done` Thought, with Mode chosen by Category:
- `feeling` → **Mirror**
- `idea` → **Challenger**
- `observation` → **Reframer**
- `learning` → **Extender**

**Additional Echo**:
An Echo the user requests after the Default has rendered, in a Mode they choose. Capped at 3 additional per Thought (4 total, one per Mode).
_Avoid_: Extra echo, alternate echo, "more angles" (UI label only).

**Mirror**:
Distills the Thought's emotional core in the user's voice and names the unsaid thing. No advice, no validation.

**Challenger**:
One probing question or one "what would break this" assertion. No flattery, no softening.

**Reframer**:
Restates the Thought from a different vantage. Surfaces the implication the user did not name.

**Extender**:
One adjacent concept or one "next question to explore". No summary of what the user learned.

## Relationships

- A **Thought** has exactly one **Enrichment** once enrichment succeeds.
- A **Thought** in status `done` has exactly one **Default Echo**, whose **Mode** is determined by its **Category**.
- A **Thought** may have up to three **Additional Echoes**, each in a distinct **Mode**.
- An **Echo** belongs to exactly one **Thought** and has exactly one **Mode**.
- **Echo** lifecycle is independent of **Thought** lifecycle: an Echo failure leaves the Thought `done` and is invisible to the user.

## Example dialogue

> **Dev:** "If a **Thought** is categorized as `feeling`, when does the **Mirror** appear?"
> **Domain expert:** "Once the **Thought** reaches `done`. A sweeper picks up `done` Thoughts that don't yet have their **Default Echo** and generates one. If generation fails three times, nothing is shown — the Thought reads as if the Echo system isn't there."
>
> **Dev:** "And if the user wants a **Challenger** on a `feeling` Thought?"
> **Domain expert:** "They tap 'more angles', pick Challenger, and we generate an **Additional Echo**. It's still an Echo — same constraints, same table, just not the Default."

## Flagged ambiguities

- "Response" was used early to describe what an Echo is — resolved: **Echo**. "Response" implied conversational reply, which the constraints (no questions outside Challenger, no advice) deliberately rule out.
- "Type" was used for both **Category** (of a Thought) and **Mode** (of an Echo) — resolved: distinct words, never interchanged.
