// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'

// Components
import TimeSeries from 'src/shared/components/TimeSeries'
import EmptyQueryView, {ErrorFormat} from 'src/shared/components/EmptyQueryView'
import ViewSwitcher from 'src/shared/components/ViewSwitcher'

// Utils
import {GlobalAutoRefresher} from 'src/utils/AutoRefresher'
import {getTimeRangeVars} from 'src/variables/utils/getTimeRangeVars'
import {getVariableAssignments} from 'src/variables/selectors'
import {getDashboardValuesStatus} from 'src/variables/selectors'
import {checkResultsLength} from 'src/shared/utils/vis'

// Types
import {
  TimeRange,
  RemoteDataState,
  TimeZone,
  AppState,
  DashboardQuery,
  VariableAssignment,
  QueryViewProperties,
  Check,
} from 'src/types'

// Selectors
import {getEndTime, getStartTime} from 'src/timeMachine/selectors/index'
import {getTimeRangeByDashboardID} from 'src/dashboards/selectors/index'

interface OwnProps {
  timeRange: TimeRange
  manualRefresh: number
  properties: QueryViewProperties
  dashboardID: string
  check: Partial<Check>
}

interface StateProps {
  endTime: number
  startTime: number
  timeZone: TimeZone
  variableAssignments: VariableAssignment[]
  variablesStatus: RemoteDataState
}

interface State {
  submitToken: number
}

type Props = OwnProps & StateProps

class RefreshingView extends PureComponent<Props, State> {
  public static defaultProps = {
    inView: true,
    manualRefresh: 0,
  }

  constructor(props) {
    super(props)

    this.state = {submitToken: 0}
  }

  public componentDidMount() {
    GlobalAutoRefresher.subscribe(this.incrementSubmitToken)
  }

  public componentWillUnmount() {
    GlobalAutoRefresher.unsubscribe(this.incrementSubmitToken)
  }

  public render() {
    const {
      check,
      endTime,
      properties,
      manualRefresh,
      startTime,
      timeZone,
    } = this.props
    const {submitToken} = this.state

    return (
      <TimeSeries
        submitToken={submitToken}
        queries={this.queries}
        key={manualRefresh}
        variables={this.variableAssignments}
        check={check}
      >
        {({
          giraffeResult,
          files,
          loading,
          errorMessage,
          isInitialFetch,
          statuses,
        }) => {
          return (
            <EmptyQueryView
              errorFormat={ErrorFormat.Tooltip}
              errorMessage={errorMessage}
              hasResults={checkResultsLength(giraffeResult)}
              loading={loading}
              isInitialFetch={isInitialFetch}
              queries={this.queries}
              fallbackNote={this.fallbackNote}
            >
              <ViewSwitcher
                check={check}
                endTime={endTime}
                files={files}
                giraffeResult={giraffeResult}
                loading={loading}
                properties={properties}
                startTime={startTime}
                statuses={statuses}
                timeZone={timeZone}
              />
            </EmptyQueryView>
          )
        }}
      </TimeSeries>
    )
  }

  private get queries(): DashboardQuery[] {
    const {properties} = this.props

    switch (properties.type) {
      case 'single-stat':
      case 'gauge':
        return [properties.queries[0]]
      default:
        return properties.queries
    }
  }

  private get variableAssignments(): VariableAssignment[] {
    const {timeRange, variableAssignments} = this.props

    return [...variableAssignments, ...getTimeRangeVars(timeRange)]
  }

  private get fallbackNote(): string {
    const {properties} = this.props

    switch (properties.type) {
      case 'check':
        return null
      default:
        const {note, showNoteWhenEmpty} = properties

        return showNoteWhenEmpty ? note : null
    }
  }

  private incrementSubmitToken = () => {
    this.setState({submitToken: Date.now()})
  }
}

const mstp = (state: AppState, ownProps: OwnProps): StateProps => {
  const {ranges} = state
  const variableAssignments = getVariableAssignments(
    state,
    ownProps.dashboardID
  )
  const timeRange = getTimeRangeByDashboardID(ranges, ownProps.dashboardID)

  const valuesStatus = getDashboardValuesStatus(state, ownProps.dashboardID)

  const timeZone = state.app.persisted.timeZone

  return {
    endTime: getEndTime(timeRange),
    startTime: getStartTime(timeRange),
    timeZone,
    variableAssignments,
    variablesStatus: valuesStatus,
  }
}

export default connect<StateProps, {}, OwnProps>(mstp)(RefreshingView)
