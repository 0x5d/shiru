Shiru is a web app for Japanese learners who want to improve their reading skills and learn new vocabulary.
Users add words & phrases to their vocab library, and shiru generates stories that use them.

Stack:
- Go for the API & backend
- React for the frontend
- Postgres for transactional data
- Elasticsearch for text indexing
- Elevenlabs for text-to-speech
- Anthropic for inference

Web App

Home Page
When users open shiru, it shows 3 dynamically generated topics for story. When they select one, it generates a story around that topic using words from the user's vocab library.

Settings
Under settings, users are able to add words to their vocab library.
They can also provide their WaniKani API key, and import the words they have unlocked there. The backend should use whatever WaniKani endpoint(s) are required for vocab import.
They may also choose to sync with WaniKani to import new words they have unlocked since the last time they synced.
Users can choose their Japanese level on a slider with 5 stops, which correspond to the JLPT levels (N5 through N1).
Users can choose story length in number of words. Default is 100 words.

Adding words
When a user adds one or more words or phrases, or imports them through WaniKani, duplicate words should be merged. The shiru backend will produce up to 3 tags for the words using Anthropic and store them. For example,
|word/phrase|tags|
|-|-|
|花|nature,city,house|
|走る|exercise,fitness,action|
|飲む|food,meal,beverages|
|家族|family,house,everyday life|
|コンピューター|technology,work,appliances|

These tags are also stored to form a set among them.

Generating stories
To generate stories, the shiru backend will receive the chosen topic and query the set of tags that the user's words form. From those tags, it will choose the 3 that are the most related to the chosen topic using LLM ranking, and query the words that have those tags. The resulting number of words shouldn't be higher than 50.

Then, the chosen words will be sent to the Anthropic API, along with the chosen topic, and the user's JLPT level, to create a story.
The story should either be shocking or funny, chosen randomly.
The story should use many of the chosen words, but does not need to include all of them.
Stories must be persisted so that the user can go back to them.

Reading stories
When a story is generated, it's sent to the web app so that the user can read it. When a user long-presses a word or phrase, its meaning should pop up in a tooltip. Meanings should come from a dictionary API. For kanji, furigana should be displayed when the user short-presses/ clicks on it. Word/phrase highlighting is required.

There is also a play button. When it's pressed, the story is read and played using elevenlabs.io's text-to-speech engine. Use a single configured voice ID for now and cache generated audio.

Search
Stories should support full text retrieval via Elasticsearch.

Out of scope for now
- Authentication and authorization.

Resolved Q&A
- Auth: out of scope for now.
- WaniKani import: use whichever endpoint(s) are needed to import user vocab.
- Duplicate vocab entries: merge duplicates.
- Tag generation: Anthropic.
- Ranking related tags to topic: LLM-based ranking.
- Story word usage: use many selected words, not necessarily all.
- Story tone: randomly choose shocking or funny.
- Meanings source: dictionary API.
- Reading interaction: highlighting required.
- TTS voice selection: one configured voice ID for now.
- TTS output: cache generated audio.
- Topic source: dynamically generated.
- Elasticsearch role: full text retrieval.

Implementation plan
- See `docs/mvp-engineering-plan.md` for concrete MVP architecture, schema, API contracts, integrations, milestones, and acceptance criteria.
