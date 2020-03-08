// Library
import {Component} from 'react'
import {isEqual} from 'lodash'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'
import {fromFlux, FromFluxResult} from '@influxdata/giraffe'

// API
import {
  runQuery,
  RunQueryResult,
  RunQuerySuccessResult,
} from 'src/shared/apis/query'

// Utils
import {checkQueryResult} from 'src/shared/utils/checkQueryResult'
import {getWindowVars} from 'src/variables/utils/getWindowVars'
import {buildVarsOption} from 'src/variables/utils/buildVarsOption'

// Constants
import {rateLimitReached, resultTooLarge} from 'src/shared/copy/notifications'

// Actions
import {notify as notifyAction} from 'src/shared/actions/notifications'

// Types
import {RemoteDataState} from 'src/types'
import {DashboardQuery} from 'src/types/dashboards'
import {AppState} from 'src/types'
import {CancelBox} from 'src/types/promises'
import {VariableAssignment} from 'src/types/ast'

interface QueriesState {
  files: string[] | null
  loading: RemoteDataState
  errorMessage: string
  isInitialFetch: boolean
  duration: number
  giraffeResult: FromFluxResult
}

interface StateProps {
  queryLink: string
}

interface OwnProps {
  queries: DashboardQuery[]
  variables?: VariableAssignment[]
  submitToken: number
  implicitSubmit?: boolean
  children: (r: QueriesState) => JSX.Element
}

interface DispatchProps {
  notify: typeof notifyAction
}

type Props = StateProps & OwnProps & DispatchProps

interface State {
  loading: RemoteDataState
  files: string[] | null
  errorMessage: string
  fetchCount: number
  duration: number
  giraffeResult: FromFluxResult
}

const defaultState = (): State => ({
  loading: RemoteDataState.NotStarted,
  files: null,
  fetchCount: 0,
  errorMessage: '',
  duration: 0,
  giraffeResult: null,
})

class TimeSeries extends Component<Props & WithRouterProps, State> {
  public static defaultProps = {
    implicitSubmit: true,
  }

  public state: State = defaultState()

  private pendingResults: Array<CancelBox<RunQueryResult>> = []

  public async componentDidMount() {
    this.reload()
  }

  public async componentDidUpdate(prevProps: Props) {
    if (this.shouldReload(prevProps)) {
      this.reload()
    }
  }

  public render() {
    const {
      giraffeResult,
      files,
      loading,
      errorMessage,
      fetchCount,
      duration,
    } = this.state

    return this.props.children({
      giraffeResult,
      files,
      loading,
      errorMessage,
      duration,
      isInitialFetch: fetchCount === 1,
    })
  }

  private reload = async () => {
    const {variables, notify} = this.props
    const queries = this.props.queries.filter(({text}) => !!text.trim())
    const orgID = this.props.params.orgID

    if (!queries.length) {
      this.setState(defaultState())

      return
    }

    this.setState({
      loading: RemoteDataState.Loading,
      fetchCount: this.state.fetchCount + 1,
      errorMessage: '',
    })

    try {
      const startTime = Date.now()

      // Cancel any existing queries
      this.pendingResults.forEach(({cancel}) => cancel())

      // Issue new queries
      this.pendingResults = queries.map(({text}) => {
        const windowVars = getWindowVars(text, variables)
        const extern = buildVarsOption([...variables, ...windowVars])

        return runQuery(orgID, text, extern)
      })

      // Wait for new queries to complete
      const results = await Promise.all(this.pendingResults.map(r => r.promise))
      const duration = Date.now() - startTime

      for (const result of results) {
        if (result.type === 'UNKNOWN_ERROR') {
          throw new Error(result.message)
        }

        if (result.type === 'RATE_LIMIT_ERROR') {
          notify(rateLimitReached(result.retryAfter))

          throw new Error(result.message)
        }

        if (result.didTruncate) {
          notify(resultTooLarge(result.bytesRead))
        }

        checkQueryResult(result.csv)
      }

      const files = (results as RunQuerySuccessResult[]).map(r => r.csv)
      const giraffeResult = fromFlux(files.join('\n\n'))

      this.setState({
        giraffeResult,
        files,
        duration,
        loading: RemoteDataState.Done,
      })
    } catch (error) {
      if (error.name === 'CancellationError') {
        return
      }

      console.error(error)

      this.setState({
        errorMessage: error.message,
        giraffeResult: null,
        loading: RemoteDataState.Error,
      })
    }
  }

  private shouldReload(prevProps: Props) {
    if (prevProps.submitToken !== this.props.submitToken) {
      return true
    }

    if (!this.props.implicitSubmit) {
      return false
    }

    if (!isEqual(prevProps.queries, this.props.queries)) {
      return true
    }

    if (!isEqual(prevProps.variables, this.props.variables)) {
      return true
    }

    return false
  }
}

const mstp = (state: AppState) => {
  const {links} = state

  return {queryLink: links.query.self}
}

const mdtp: DispatchProps = {
  notify: notifyAction,
}

export default connect<StateProps, {}, OwnProps>(
  mstp,
  mdtp
)(withRouter(TimeSeries))
