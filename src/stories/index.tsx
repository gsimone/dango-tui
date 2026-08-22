import { launchFixtureApp } from "../runtime.tsx"
import { isFixtureStoryId } from "../data/fixtures.ts"

const requestedStoryId = process.env.STACKS_STORY
const initialStoryId = isFixtureStoryId(requestedStoryId) ? requestedStoryId : "mixed"

await launchFixtureApp({ mode: "stories", initialStoryId })
