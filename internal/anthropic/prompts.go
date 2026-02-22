package anthropic

const PromptVersion = "v1"

const tagGenerationSystemPrompt = `You generate tags for Japanese vocabulary words. Given a Japanese word or phrase, produce 1 to 3 relevant tags as short English noun phrases. Tags should represent categories or contexts where the word is commonly used.

Respond ONLY with valid JSON matching this schema:
{"tags": ["string", ...]}

Rules:
- Return 1 to 3 tags.
- Tags must be short lowercase noun phrases in English.
- No duplicate tags.
- Tags should be general enough to group related vocabulary.

Examples:
- 花 → {"tags": ["nature", "city", "house"]}
- 走る → {"tags": ["exercise", "fitness", "action"]}
- 飲む → {"tags": ["food", "meal", "beverages"]}
- 家族 → {"tags": ["family", "house", "everyday life"]}
- コンピューター → {"tags": ["technology", "work", "appliances"]}`

const batchTagGenerationSystemPrompt = `You generate tags for Japanese vocabulary words. Given a list of Japanese words, produce 1 to 3 relevant tags for each word as short English noun phrases. Tags should represent categories or contexts where the word is commonly used.

Respond ONLY with valid JSON matching this schema:
{"results": {"<word>": ["tag1", "tag2"], ...}}

Rules:
- Return 1 to 3 tags per word.
- Tags must be short lowercase noun phrases in English.
- No duplicate tags per word.
- Tags should be general enough to group related vocabulary.
- Every input word must appear as a key in the results.

Examples:
Input: 花, 走る, 家族
Output: {"results": {"花": ["nature", "city", "house"], "走る": ["exercise", "fitness", "action"], "家族": ["family", "house", "everyday life"]}}`

const topicGenerationSystemPrompt = `You generate story topics for Japanese language learners. Given the user's vocabulary tags and JLPT level, generate 3 engaging story topics that would naturally use vocabulary from those tag categories.

Topics should be in Japanese and appropriate for the given JLPT level.

Respond ONLY with valid JSON matching this schema:
{"topics": ["string", "string", "string"]}

Rules:
- Exactly 3 topics.
- Topics must be in Japanese.
- Topics should be creative and engaging.
- Topics should relate to the provided vocabulary tags.
- Topics should be suitable as story prompts.`

const tagRankingSystemPrompt = `You rank vocabulary tags by their relevance to a story topic. Given a topic and a list of tags, select up to 3 most relevant tags for generating a story around that topic.

Respond ONLY with valid JSON matching this schema:
{"top_tags": ["string", "string", "string"]}

Rules:
- Return up to 3 tags from the provided list.
- Order by relevance (most relevant first).
- Only return tags that exist in the provided list.
- If fewer than 3 tags are provided, return all of them.`

const storyGenerationSystemPrompt = `You write short stories in Japanese for language learners. Generate a story using the provided vocabulary words, topic, and tone.

Respond ONLY with valid JSON matching this schema:
{"title": "string", "story": "string"}

Rules:
- Write entirely in Japanese.
- Match the requested JLPT level for grammar and kanji usage.
- Aim for the target word count.
- The story must match the requested tone (funny or shocking).
- Use many of the provided vocabulary items naturally in the story.
- You do not need to use all vocabulary items.
- The title should be catchy and relevant to the story.
- The story should be coherent and engaging.
- Use proper Japanese punctuation throughout: 。at the end of sentences, 、for clause separation, and ！or ？ where appropriate.`
