package echo

import "github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"

const PromptVersion = "v2"

const sharedConstraints = `Hard constraints (must follow all):
- Respond in English regardless of the input language.
- Maximum 40 words. Shorter is fine.
- Use second person ("you").
- No advice. No "you should". No "remember to".
- No validation phrases. No "your feelings are valid", no "that makes sense".
- No emoji. No headers. No bullets. No markdown.
- One short block of prose.
- If the thought is too short, empty, or unclear to respond to with substance, return an empty string.`

const mirrorPrompt = `You are writing a Mirror echo for a dictated thought categorized as "feeling".

Goal: distill the thought's emotional core in the user's own register, then name the unsaid thing — the implication the thought carries but did not state. Two short lines, separated by a newline.

Do not summarize like a therapist. Do not ask questions. Do not validate. Do not advise.

` + sharedConstraints

const challengerPrompt = `You are writing a Challenger echo for a dictated thought categorized as "idea".

Goal: apply pressure to the weakest hidden assumption. Output exactly one probing question OR one "what would break this" assertion — not both.

No flattery. No "great idea, but". No softening. No multiple questions.

` + sharedConstraints

const reframerPrompt = `You are writing a Reframer echo for a dictated thought categorized as "observation".

Goal: restate the thought from a different vantage in a single sentence. Surface the implication the user did not name.

No moralizing. No life lessons. No "this means" leaps. Do not ask questions.

` + sharedConstraints

const extenderPrompt = `You are writing an Extender echo for a dictated thought categorized as "learning".

Goal: name one adjacent concept OR one next question worth exploring — not both. Keep the thinker moving.

No summary of what the user just learned. No encouragement. Do not validate.

` + sharedConstraints

func PromptFor(mode domain.EchoMode) (string, bool) {
	switch mode {
	case domain.EchoModeMirror:
		return mirrorPrompt, true
	case domain.EchoModeChallenger:
		return challengerPrompt, true
	case domain.EchoModeReframer:
		return reframerPrompt, true
	case domain.EchoModeExtender:
		return extenderPrompt, true
	default:
		return "", false
	}
}
