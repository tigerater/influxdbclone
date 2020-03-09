// Libraries
import React, {PureComponent} from 'react'
import _ from 'lodash'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {Page, IconFont, Sort} from '@influxdata/clockface'
import TaskRunsList from 'src/tasks/components/TaskRunsList'
import PageTitleWithOrg from 'src/shared/components/PageTitleWithOrg'

// Types
import {AppState} from 'src/types'
import {RemoteDataState} from 'src/types'
import {Run as APIRun, Task} from '@influxdata/influx'
import {
  SpinnerContainer,
  TechnoSpinner,
  Button,
  ComponentColor,
} from '@influxdata/clockface'

// Actions
import {getRuns, runTask} from 'src/tasks/actions'

// Utils
import {pageTitleSuffixer} from 'src/shared/utils/pageTitles'

// Types
import {SortTypes} from 'src/shared/utils/sort'

export interface Run extends APIRun {
  duration: string
}

interface OwnProps {
  params: {id: string}
}

interface DispatchProps {
  getRuns: typeof getRuns
  onRunTask: typeof runTask
}

interface StateProps {
  runs: Run[]
  runStatus: RemoteDataState
  currentTask: Task
}

type Props = OwnProps & DispatchProps & StateProps

interface State {
  sortKey: SortKey
  sortDirection: Sort
  sortType: SortTypes
}

type SortKey = keyof Run

class TaskRunsPage extends PureComponent<Props & WithRouterProps, State> {
  constructor(props) {
    super(props)
    this.state = {
      sortKey: 'scheduledFor',
      sortDirection: Sort.Descending,
      sortType: SortTypes.Date,
    }
  }

  public render() {
    const {params, runs} = this.props
    const {sortKey, sortDirection, sortType} = this.state

    return (
      <SpinnerContainer
        loading={this.props.runStatus}
        spinnerComponent={<TechnoSpinner />}
      >
        <Page titleTag={pageTitleSuffixer(['Task Runs'])}>
          <Page.Header fullWidth={false}>
            <Page.Header.Left>
              <PageTitleWithOrg title={this.title} />
            </Page.Header.Left>
            <Page.Header.Right>
              <Button
                onClick={this.handleEditTask}
                text="Edit Task"
                color={ComponentColor.Primary}
              />
              <Button
                onClick={this.handleRunTask}
                text="Run Task"
                icon={IconFont.Play}
              />
            </Page.Header.Right>
          </Page.Header>
          <Page.Contents fullWidth={false} scrollable={true}>
            <TaskRunsList
              taskID={params.id}
              runs={runs}
              sortKey={sortKey}
              sortDirection={sortDirection}
              sortType={sortType}
              onClickColumn={this.handleClickColumn}
            />
          </Page.Contents>
        </Page>
      </SpinnerContainer>
    )
  }

  public componentDidMount() {
    this.props.getRuns(this.props.params.id)
  }

  private handleClickColumn = (nextSort: Sort, sortKey: SortKey) => {
    let sortType = SortTypes.String

    if (sortKey !== 'status') {
      sortType = SortTypes.Date
    }

    this.setState({sortKey, sortDirection: nextSort, sortType})
  }

  private get title() {
    const {currentTask} = this.props

    if (currentTask) {
      return `${currentTask.name} - Runs`
    }
    return 'Runs'
  }

  private handleRunTask = async () => {
    const {onRunTask, params, getRuns} = this.props
    await onRunTask(params.id)
    getRuns(params.id)
  }

  private handleEditTask = () => {
    const {
      router,
      currentTask,
      params: {orgID},
    } = this.props

    router.push(`/orgs/${orgID}/tasks/${currentTask.id}`)
  }
}

const mstp = (state: AppState): StateProps => {
  const {tasks} = state

  return {
    runs: tasks.runs,
    runStatus: tasks.runStatus,
    currentTask: tasks.currentTask,
  }
}

const mdtp: DispatchProps = {
  getRuns: getRuns,
  onRunTask: runTask,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<OwnProps>(TaskRunsPage))
