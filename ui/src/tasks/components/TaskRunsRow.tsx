// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import moment from 'moment'

// Components
import {Overlay, IndexList} from '@influxdata/clockface'
import RunLogsOverlay from 'src/tasks/components/RunLogsList'

// Actions
import {getLogs} from 'src/tasks/actions'

// Types
import {ComponentSize, ComponentColor, Button} from '@influxdata/clockface'
import {LogEvent} from '@influxdata/influx'
import {AppState} from 'src/types'
import {DEFAULT_TIME_FORMAT} from 'src/shared/constants'
import {Run} from 'src/tasks/components/TaskRunsPage'

interface OwnProps {
  taskID: string
  run: Run
}

interface DispatchProps {
  getLogs: typeof getLogs
}

interface StateProps {
  logs: LogEvent[]
}

type Props = OwnProps & DispatchProps & StateProps

interface State {
  isImportOverlayVisible: boolean
}

class TaskRunsRow extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)

    this.state = {
      isImportOverlayVisible: false,
    }
  }

  public render() {
    const {run} = this.props
    return (
      <>
        <IndexList.Row>
          <IndexList.Cell>{run.status}</IndexList.Cell>
          <IndexList.Cell>
            {this.dateTimeString(run.scheduledFor)}
          </IndexList.Cell>
          <IndexList.Cell>{this.dateTimeString(run.startedAt)}</IndexList.Cell>
          <IndexList.Cell>{run.duration}</IndexList.Cell>
          <IndexList.Cell>
            <Button
              key={run.id}
              size={ComponentSize.ExtraSmall}
              color={ComponentColor.Default}
              text="View Logs"
              onClick={this.handleToggleOverlay}
            />
            {this.renderLogOverlay}
          </IndexList.Cell>
        </IndexList.Row>
      </>
    )
  }

  private dateTimeString(dt: Date): string {
    if (!dt) {
      return ''
    }
    const newdate = new Date(dt)
    const formatted = moment(newdate).format(DEFAULT_TIME_FORMAT)

    return formatted
  }

  private handleToggleOverlay = async () => {
    const {taskID, run, getLogs} = this.props
    await getLogs(taskID, run.id)

    this.setState({isImportOverlayVisible: !this.state.isImportOverlayVisible})
  }

  private get renderLogOverlay(): JSX.Element {
    const {isImportOverlayVisible} = this.state
    const {logs} = this.props

    return (
      <Overlay visible={isImportOverlayVisible}>
        <RunLogsOverlay
          onDismissOverlay={this.handleToggleOverlay}
          logs={logs}
        />
      </Overlay>
    )
  }
}

const mstp = (state: AppState): StateProps => {
  const {
    tasks: {logs},
  } = state
  return {logs}
}

const mdtp: DispatchProps = {getLogs: getLogs}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(TaskRunsRow)
