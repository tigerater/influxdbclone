// APIs
import {client} from 'src/utils/api'

// Utils
import {getMinDurationFromAST} from 'src/shared/utils/getMinDurationFromAST'

// Constants
import {DEFAULT_DURATION_MS, WINDOW_PERIOD} from 'src/shared/constants'

// Types
import {InfluxLanguage} from 'src/types/v2/dashboards'

const DESIRED_POINTS_PER_GRAPH = 360
const FALLBACK_WINDOW_PERIOD = 15000

export async function renderQuery(
  query: string,
  type: InfluxLanguage,
  variables: {[name: string]: string}
): Promise<string> {
  if (type === InfluxLanguage.InfluxQL) {
    // We don't support template variables / macros in InfluxQL yet, so this is
    // a no-op
    return query
  }

  const {imports, body} = await extractImports(query)
  let variableDeclarations = formatVariables(variables, query)

  if (query.includes(WINDOW_PERIOD)) {
    const ast = await client.queries.ast(`${variableDeclarations}\n\n${query}`)

    let windowPeriod: number

    try {
      windowPeriod = getWindowInterval(getMinDurationFromAST(ast))
    } catch {
      windowPeriod = FALLBACK_WINDOW_PERIOD
    }

    variableDeclarations += `\n${WINDOW_PERIOD} = ${windowPeriod}ms`
  }

  return `${imports}\n\n${variableDeclarations}\n\n${body}`
}

async function extractImports(
  query: string
): Promise<{imports: string; body: string}> {
  const ast = await client.queries.ast(query)
  const {imports, body} = ast.files[0]
  const importStatements = (imports || [])
    .map(i => i.location.source)
    .join('\n')
  const bodyStatements = (body || []).map(b => b.location.source).join('\n')
  return {imports: importStatements, body: bodyStatements}
}

function formatVariables(
  variables: {[name: string]: string},
  query: string
): string {
  return Object.entries(variables)
    .filter(([key]) => query.includes(key))
    .map(([key, value]) => `${key} = ${value}`)
    .join('\n')
}

function getWindowInterval(durationMilliseconds: number = DEFAULT_DURATION_MS) {
  return Math.round(durationMilliseconds / DESIRED_POINTS_PER_GRAPH)
}
