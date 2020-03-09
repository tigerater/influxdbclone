// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'
import {get, isEmpty} from 'lodash'

// Selectors
import {getSaveableView} from 'src/timeMachine/selectors'
import {getOrg} from 'src/organizations/selectors'
import {getAll} from 'src/resources/selectors'

// Components
import {Form, Input, Button, Grid} from '@influxdata/clockface'
import {ErrorHandling} from 'src/shared/decorators/errors'
import DashboardsDropdown from 'src/dataExplorer/components/DashboardsDropdown'

// Constants
import {cellAddFailed, cellAdded} from 'src/shared/copy/notifications'
import {
  DashboardTemplate,
  DEFAULT_DASHBOARD_NAME,
  DEFAULT_CELL_NAME,
} from 'src/dashboards/constants'

// Actions
import {getDashboards} from 'src/dashboards/actions/thunks'
import {createCellWithView} from 'src/cells/actions/thunks'
import {postDashboard} from 'src/client'
import {notify} from 'src/shared/actions/notifications'

// Types
import {AppState, Dashboard, View, ResourceType} from 'src/types'
import {
  Columns,
  InputType,
  ButtonType,
  ComponentColor,
  ComponentStatus,
} from '@influxdata/clockface'

interface State {
  targetDashboardIDs: string[]
  cellName: string
  isNameDashVisible: boolean
  newDashboardName: string
}

interface StateProps {
  dashboards: Dashboard[]
  view: View
  orgID: string
}

interface DispatchProps {
  onGetDashboards: typeof getDashboards
  onCreateCellWithView: typeof createCellWithView
  notify: typeof notify
}

interface OwnProps {
  dismiss: () => void
}

type Props = StateProps & DispatchProps & OwnProps

@ErrorHandling
class SaveAsCellForm extends PureComponent<Props, State> {
  public state: State = {
    targetDashboardIDs: [],
    cellName: '',
    isNameDashVisible: false,
    newDashboardName: DEFAULT_DASHBOARD_NAME,
  }

  public componentDidMount() {
    const {onGetDashboards} = this.props
    onGetDashboards()
  }

  public render() {
    const {dismiss, dashboards} = this.props
    const {
      cellName,
      isNameDashVisible,
      targetDashboardIDs,
      newDashboardName,
    } = this.state
    return (
      <Form onSubmit={this.handleSubmit}>
        <Grid>
          <Grid.Row>
            <Grid.Column widthXS={Columns.Twelve}>
              <Form.Element label="Target Dashboard(s)">
                <DashboardsDropdown
                  onSelect={this.handleSelectDashboardID}
                  selectedIDs={targetDashboardIDs}
                  dashboards={dashboards}
                  newDashboardName={newDashboardName}
                />
              </Form.Element>
            </Grid.Column>
            {isNameDashVisible && this.nameDashboard}
            <Grid.Column widthXS={Columns.Twelve}>
              <Form.Element label="Cell Name">
                <Input
                  type={InputType.Text}
                  placeholder="Add optional cell name"
                  name="cellName"
                  value={cellName}
                  onChange={this.handleChangeCellName}
                  testID="save-as-dashboard-cell--cell-name"
                />
              </Form.Element>
            </Grid.Column>
            <Grid.Column widthXS={Columns.Twelve}>
              <Form.Footer>
                <Button
                  text="Cancel"
                  onClick={dismiss}
                  titleText="Cancel save"
                  type={ButtonType.Button}
                  testID="save-as-dashboard-cell--cancel"
                />
                <Button
                  text="Save as Dashboard Cell"
                  testID="save-as-dashboard-cell--submit"
                  color={ComponentColor.Success}
                  type={ButtonType.Submit}
                  onClick={this.handleSubmit}
                  status={
                    this.isFormValid
                      ? ComponentStatus.Default
                      : ComponentStatus.Disabled
                  }
                />
              </Form.Footer>
            </Grid.Column>
          </Grid.Row>
        </Grid>
      </Form>
    )
  }

  private get nameDashboard(): JSX.Element {
    const {newDashboardName} = this.state
    return (
      <Grid.Column widthXS={Columns.Twelve}>
        <Form.Element label="New Dashboard Name">
          <Input
            type={InputType.Text}
            placeholder="Add dashboard name"
            name="dashboardName"
            value={newDashboardName}
            onChange={this.handleChangeDashboardName}
            testID="save-as-dashboard-cell--dashboard-name"
          />
        </Form.Element>
      </Grid.Column>
    )
  }

  private get isFormValid(): boolean {
    const {targetDashboardIDs} = this.state
    return !isEmpty(targetDashboardIDs)
  }

  private handleSubmit = () => {
    const {onCreateCellWithView, dashboards, view, dismiss, notify} = this.props
    const {targetDashboardIDs} = this.state

    const cellName = this.state.cellName || DEFAULT_CELL_NAME
    const newDashboardName =
      this.state.newDashboardName || DEFAULT_DASHBOARD_NAME

    const viewWithProps: View = {...view, name: cellName}

    try {
      targetDashboardIDs.forEach(dashID => {
        let targetDashboardName = ''
        try {
          if (dashID === DashboardTemplate.id) {
            targetDashboardName = newDashboardName
            this.handleCreateDashboardWithView(newDashboardName, viewWithProps)
          } else {
            const selectedDashboard = dashboards.find(d => d.id === dashID)
            targetDashboardName = selectedDashboard.name
            onCreateCellWithView(selectedDashboard.id, viewWithProps)
          }
          notify(cellAdded(cellName, targetDashboardName))
        } catch {
          notify(cellAddFailed(cellName, targetDashboardName))
        }
      })
    } finally {
      this.resetForm()
      dismiss()
    }
  }

  private handleCreateDashboardWithView = async (
    dashboardName: string,
    view: View
  ): Promise<void> => {
    const {onCreateCellWithView, orgID} = this.props
    try {
      const newDashboard = {
        orgID,
        name: dashboardName || DEFAULT_DASHBOARD_NAME,
        cells: [],
      }

      const resp = await postDashboard({data: newDashboard})

      if (resp.status !== 201) {
        throw new Error(resp.data.message)
      }

      onCreateCellWithView(resp.data.id, view)
    } catch (error) {
      console.error(error)
    }
  }

  private resetForm() {
    this.setState({
      targetDashboardIDs: [],
      cellName: '',
      isNameDashVisible: false,
      newDashboardName: DEFAULT_DASHBOARD_NAME,
    })
  }

  private handleSelectDashboardID = (
    selectedIDs: string[],
    value: Dashboard
  ) => {
    if (value.id === DashboardTemplate.id) {
      this.setState({
        isNameDashVisible: selectedIDs.includes(DashboardTemplate.id),
      })
    }
    this.setState({targetDashboardIDs: selectedIDs})
  }

  private handleChangeDashboardName = (e: ChangeEvent<HTMLInputElement>) => {
    this.setState({newDashboardName: e.target.value})
  }

  private handleChangeCellName = (e: ChangeEvent<HTMLInputElement>) => {
    this.setState({cellName: e.target.value})
  }
}

const mstp = (state: AppState): StateProps => {
  const view = getSaveableView(state)
  const org = getOrg(state)
  const dashboards = getAll<Dashboard>(state, ResourceType.Dashboards)

  return {dashboards, view, orgID: get(org, 'id', '')}
}

const mdtp: DispatchProps = {
  onGetDashboards: getDashboards,
  onCreateCellWithView: createCellWithView,
  notify,
}

export default connect<StateProps, DispatchProps>(
  mstp,
  mdtp
)(SaveAsCellForm)
