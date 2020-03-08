// Libraries
import {Dispatch} from 'redux'
import {replace, push} from 'react-router-redux'

// APIs
import {
  createDashboard as createDashboardAJAX,
  getDashboard as getDashboardAJAX,
  getDashboards as getDashboardsAJAX,
  deleteDashboard as deleteDashboardAJAX,
  updateDashboard as updateDashboardAJAX,
  updateCells as updateCellsAJAX,
  addCell as addCellAJAX,
  deleteCell as deleteCellAJAX,
  getView as getViewAJAX,
  updateView as updateViewAJAX,
} from 'src/dashboards/apis'
import {createDashboardFromTemplate as createDashboardFromTemplateAJAX} from 'src/templates/api'

// Actions
import {
  notify,
  PublishNotificationAction,
} from 'src/shared/actions/notifications'
import {
  deleteTimeRange,
  updateTimeRangeFromQueryParams,
  DeleteTimeRangeAction,
} from 'src/dashboards/actions/ranges'
import {setView, SetViewAction, setViews} from 'src/dashboards/actions/views'
import {
  getVariables,
  refreshVariableValues,
  selectValue,
} from 'src/variables/actions'
import {setExportTemplate} from 'src/templates/actions'
import {checkDashboardLimits} from 'src/cloud/actions/limits'

// Utils
import {filterUnusedVars} from 'src/shared/utils/filterUnusedVars'
import {
  extractVariablesList,
  getHydratedVariables,
} from 'src/variables/selectors'
import {getViewsForDashboard} from 'src/dashboards/selectors'
import {
  getNewDashboardCell,
  getClonedDashboardCell,
} from 'src/dashboards/utils/cellGetters'
import {dashboardToTemplate} from 'src/shared/utils/resourceToTemplate'
import {client} from 'src/utils/api'
import {exportVariables} from 'src/variables/utils/exportVariables'
import {getSaveableView} from 'src/timeMachine/selectors'
import {incrementCloneName} from 'src/utils/naming'
import {isLimitError} from 'src/cloud/utils/limits'

// Constants
import * as copy from 'src/shared/copy/notifications'
import {DEFAULT_DASHBOARD_NAME} from 'src/dashboards/constants/index'

// Types
import {RemoteDataState} from 'src/types'
import {CreateCell, ILabel} from '@influxdata/influx'
import {
  Dashboard,
  NewView,
  Cell,
  GetState,
  View,
  DashboardTemplate,
} from 'src/types'

export enum ActionTypes {
  SetDashboards = 'SET_DASHBOARDS',
  SetDashboard = 'SET_DASHBOARD',
  RemoveDashboard = 'REMOVE_DASHBOARD',
  DeleteDashboardFailed = 'DELETE_DASHBOARD_FAILED',
  EditDashboard = 'EDIT_DASHBOARD',
  RemoveCell = 'REMOVE_CELL',
  AddDashboardLabels = 'ADD_DASHBOARD_LABELS',
  RemoveDashboardLabels = 'REMOVE_DASHBOARD_LABELS',
}

export type Action =
  | SetDashboardsAction
  | RemoveDashboardAction
  | SetDashboardAction
  | EditDashboardAction
  | RemoveCellAction
  | SetViewAction
  | DeleteTimeRangeAction
  | DeleteDashboardFailedAction
  | AddDashboardLabelsAction
  | RemoveDashboardLabelsAction

interface RemoveCellAction {
  type: ActionTypes.RemoveCell
  payload: {
    dashboard: Dashboard
    cell: Cell
  }
}

interface EditDashboardAction {
  type: ActionTypes.EditDashboard
  payload: {
    dashboard: Dashboard
  }
}

interface SetDashboardsAction {
  type: ActionTypes.SetDashboards
  payload: {
    status: RemoteDataState
    list: Dashboard[]
  }
}

interface RemoveDashboardAction {
  type: ActionTypes.RemoveDashboard
  payload: {
    id: string
  }
}

interface DeleteDashboardFailedAction {
  type: ActionTypes.DeleteDashboardFailed
  payload: {
    dashboard: Dashboard
  }
}

interface SetDashboardAction {
  type: ActionTypes.SetDashboard
  payload: {
    dashboard: Dashboard
  }
}

interface AddDashboardLabelsAction {
  type: ActionTypes.AddDashboardLabels
  payload: {
    dashboardID: string
    labels: ILabel[]
  }
}

interface RemoveDashboardLabelsAction {
  type: ActionTypes.RemoveDashboardLabels
  payload: {
    dashboardID: string
    labels: ILabel[]
  }
}

// Action Creators

export const editDashboard = (dashboard: Dashboard): EditDashboardAction => ({
  type: ActionTypes.EditDashboard,
  payload: {dashboard},
})

export const setDashboards = (
  status: RemoteDataState,
  list?: Dashboard[]
): SetDashboardsAction => ({
  type: ActionTypes.SetDashboards,
  payload: {
    status,
    list,
  },
})

export const setDashboard = (dashboard: Dashboard): SetDashboardAction => ({
  type: ActionTypes.SetDashboard,
  payload: {dashboard},
})

export const removeDashboard = (id: string): RemoveDashboardAction => ({
  type: ActionTypes.RemoveDashboard,
  payload: {id},
})

export const deleteDashboardFailed = (
  dashboard: Dashboard
): DeleteDashboardFailedAction => ({
  type: ActionTypes.DeleteDashboardFailed,
  payload: {dashboard},
})

export const removeCell = (
  dashboard: Dashboard,
  cell: Cell
): RemoveCellAction => ({
  type: ActionTypes.RemoveCell,
  payload: {dashboard, cell},
})

export const addDashboardLabels = (
  dashboardID: string,
  labels: ILabel[]
): AddDashboardLabelsAction => ({
  type: ActionTypes.AddDashboardLabels,
  payload: {dashboardID, labels},
})

export const removeDashboardLabels = (
  dashboardID: string,
  labels: ILabel[]
): RemoveDashboardLabelsAction => ({
  type: ActionTypes.RemoveDashboardLabels,
  payload: {dashboardID, labels},
})

// Thunks

export const createDashboard = () => async (
  dispatch,
  getState: GetState
): Promise<void> => {
  try {
    const {
      orgs: {org},
    } = getState()

    const newDashboard = {
      name: DEFAULT_DASHBOARD_NAME,
      cells: [],
      orgID: org.id,
    }

    const data = await createDashboardAJAX(newDashboard)
    dispatch(push(`/orgs/${org.id}/dashboards/${data.id}`))
    dispatch(checkDashboardLimits())
  } catch (error) {
    console.error(error)

    if (isLimitError(error)) {
      dispatch(notify(copy.resourceLimitReached('dashboards')))
    } else {
      dispatch(notify(copy.dashboardCreateFailed()))
    }
  }
}

export const cloneDashboard = (dashboard: Dashboard) => async (
  dispatch,
  getState: GetState
): Promise<void> => {
  try {
    const {
      orgs: {org},
      dashboards,
    } = getState()

    const allDashboardNames = dashboards.list.map(d => d.name)

    const clonedName = incrementCloneName(allDashboardNames, dashboard.name)

    const data = await client.dashboards.clone(dashboard.id, clonedName, org.id)

    dispatch(checkDashboardLimits())
    dispatch(push(`/orgs/${org.id}/dashboards/${data.id}`))
  } catch (error) {
    console.error(error)
    if (isLimitError(error)) {
      dispatch(notify(copy.resourceLimitReached('dashboards')))
    } else {
      dispatch(notify(copy.dashboardCreateFailed()))
    }
  }
}

export const getDashboardsAsync = () => async (
  dispatch: Dispatch<Action>,
  getState: GetState
): Promise<Dashboard[]> => {
  try {
    const {
      orgs: {org},
    } = getState()

    dispatch(setDashboards(RemoteDataState.Loading))
    const dashboards = await getDashboardsAJAX(org.id)
    dispatch(setDashboards(RemoteDataState.Done, dashboards))

    return dashboards
  } catch (error) {
    dispatch(setDashboards(RemoteDataState.Error))
    console.error(error)
    throw error
  }
}

export const createDashboardFromTemplate = (
  template: DashboardTemplate
) => async (dispatch, getState: GetState) => {
  try {
    const {
      orgs: {org},
    } = getState()

    await createDashboardFromTemplateAJAX(template, org.id)

    const dashboards = await getDashboardsAJAX(org.id)

    dispatch(setDashboards(RemoteDataState.Done, dashboards))
    dispatch(notify(copy.importDashboardSucceeded()))
    dispatch(checkDashboardLimits())
  } catch (error) {
    if (isLimitError(error)) {
      dispatch(notify(copy.resourceLimitReached('dashboards')))
    } else {
      dispatch(notify(copy.importDashboardFailed(error)))
    }
  }
}

export const deleteDashboardAsync = (dashboard: Dashboard) => async (
  dispatch
): Promise<void> => {
  dispatch(removeDashboard(dashboard.id))
  dispatch(deleteTimeRange(dashboard.id))

  try {
    await deleteDashboardAJAX(dashboard)
    dispatch(notify(copy.dashboardDeleted(dashboard.name)))
    dispatch(checkDashboardLimits())
  } catch (error) {
    dispatch(
      notify(copy.dashboardDeleteFailed(dashboard.name, error.data.message))
    )

    dispatch(deleteDashboardFailed(dashboard))
  }
}

export const refreshDashboardVariableValues = (
  dashboard: Dashboard,
  nextViews: View[]
) => (dispatch, getState: GetState) => {
  const variables = extractVariablesList(getState())
  const variablesInUse = filterUnusedVars(variables, nextViews)

  return dispatch(refreshVariableValues(dashboard.id, variablesInUse))
}

export const getDashboardAsync = (dashboardID: string) => async (
  dispatch
): Promise<void> => {
  try {
    // Fetch the dashboard and all variables a user has access to
    const [dashboard] = await Promise.all([
      getDashboardAJAX(dashboardID),
      dispatch(getVariables()),
    ])

    // Fetch all the views in use on the dashboard
    const views = await Promise.all(
      dashboard.cells.map(cell => getViewAJAX(dashboard.id, cell.id))
    )

    dispatch(setViews(RemoteDataState.Done, views))

    // Ensure the values for the variables in use on the dashboard are populated
    await dispatch(refreshDashboardVariableValues(dashboard, views))

    // Now that all the necessary state has been loaded, set the dashboard
    dispatch(setDashboard(dashboard))
  } catch {
    dispatch(replace(`/dashboards`))
    dispatch(notify(copy.dashboardGetFailed(dashboardID)))

    return
  }

  dispatch(updateTimeRangeFromQueryParams(dashboardID))
}

export const updateDashboardAsync = (dashboard: Dashboard) => async (
  dispatch: Dispatch<Action | PublishNotificationAction>
): Promise<void> => {
  try {
    const updatedDashboard = await updateDashboardAJAX(dashboard)
    dispatch(editDashboard(updatedDashboard))
  } catch (error) {
    console.error(error)
    dispatch(notify(copy.dashboardUpdateFailed()))
  }
}

export const createCellWithView = (
  dashboard: Dashboard,
  view: NewView,
  clonedCell?: Cell
) => async (dispatch, getState: GetState): Promise<void> => {
  try {
    const cell: CreateCell = getNewDashboardCell(dashboard, clonedCell)

    // Create the cell
    const createdCell = await addCellAJAX(dashboard.id, cell)

    // Create the view and associate it with the cell
    const newView = await updateViewAJAX(dashboard.id, createdCell.id, view)

    // Update the dashboard with the new cell
    let updatedDashboard: Dashboard = {
      ...dashboard,
      cells: [...dashboard.cells, createdCell],
    }

    updatedDashboard = await updateDashboardAJAX(dashboard)

    // Refresh variables in use on dashboard
    const views = [...getViewsForDashboard(getState(), dashboard.id), newView]

    await dispatch(refreshDashboardVariableValues(dashboard, views))

    dispatch(setView(createdCell.id, newView, RemoteDataState.Done))
    dispatch(editDashboard(updatedDashboard))
  } catch {
    notify(copy.cellAddFailed())
  }
}

export const updateView = (dashboard: Dashboard, view: View) => async (
  dispatch,
  getState: GetState
) => {
  const cellID = view.cellID

  try {
    const newView = await updateViewAJAX(dashboard.id, cellID, view)

    const views = getViewsForDashboard(getState(), dashboard.id)

    views.splice(views.findIndex(v => v.id === newView.id), 1, newView)

    await dispatch(refreshDashboardVariableValues(dashboard, views))

    dispatch(setView(cellID, newView, RemoteDataState.Done))
  } catch (e) {
    console.error(e)
    dispatch(notify(copy.cellUpdateFailed()))
    dispatch(setView(cellID, null, RemoteDataState.Error))
  }
}

export const updateCellsAsync = (dashboard: Dashboard, cells: Cell[]) => async (
  dispatch: Dispatch<Action>
): Promise<void> => {
  try {
    const updatedCells = await updateCellsAJAX(dashboard.id, cells)
    const updatedDashboard = {
      ...dashboard,
      cells: updatedCells,
    }

    dispatch(setDashboard(updatedDashboard))
  } catch (error) {
    console.error(error)
  }
}

export const deleteCellAsync = (dashboard: Dashboard, cell: Cell) => async (
  dispatch,
  getState: GetState
): Promise<void> => {
  try {
    const views = getViewsForDashboard(getState(), dashboard.id).filter(
      view => view.cellID !== cell.id
    )

    await Promise.all([
      deleteCellAJAX(dashboard.id, cell),
      dispatch(refreshDashboardVariableValues(dashboard, views)),
    ])

    dispatch(removeCell(dashboard, cell))
    dispatch(notify(copy.cellDeleted()))
  } catch (error) {
    console.error(error)
  }
}

export const copyDashboardCellAsync = (
  dashboard: Dashboard,
  cell: Cell
) => async (
  dispatch: Dispatch<Action | PublishNotificationAction>
): Promise<void> => {
  try {
    const clonedCell = getClonedDashboardCell(dashboard, cell)
    const updatedDashboard = {
      ...dashboard,
      cells: [...dashboard.cells, clonedCell],
    }

    dispatch(setDashboard(updatedDashboard))
    dispatch(notify(copy.cellAdded()))
  } catch (error) {
    console.error(error)
  }
}

export const addDashboardLabelsAsync = (
  dashboardID: string,
  labels: ILabel[]
) => async (dispatch: Dispatch<Action | PublishNotificationAction>) => {
  try {
    const newLabels = await client.dashboards.addLabels(
      dashboardID,
      labels.map(l => l.id)
    )

    dispatch(addDashboardLabels(dashboardID, newLabels))
  } catch (error) {
    console.error(error)
    dispatch(notify(copy.addDashboardLabelFailed()))
  }
}

export const removeDashboardLabelsAsync = (
  dashboardID: string,
  labels: ILabel[]
) => async (dispatch: Dispatch<Action | PublishNotificationAction>) => {
  try {
    await client.dashboards.removeLabels(dashboardID, labels.map(l => l.id))

    dispatch(removeDashboardLabels(dashboardID, labels))
  } catch (error) {
    console.error(error)
    dispatch(notify(copy.removedDashboardLabelFailed()))
  }
}

export const selectVariableValue = (
  dashboardID: string,
  variableID: string,
  value: string
) => async (dispatch, getState: GetState): Promise<void> => {
  const variables = getHydratedVariables(getState(), dashboardID)
  const dashboard = getState().dashboards.list.find(d => d.id === dashboardID)

  dispatch(selectValue(dashboardID, variableID, value))

  await dispatch(refreshVariableValues(dashboard.id, variables))
}

export const convertToTemplate = (dashboardID: string) => async (
  dispatch,
  getState: GetState
): Promise<void> => {
  try {
    dispatch(setExportTemplate(RemoteDataState.Loading))
    const {
      orgs: {org},
    } = getState()
    const dashboard = await getDashboardAJAX(dashboardID)
    const pendingViews = dashboard.cells.map(c =>
      getViewAJAX(dashboardID, c.id)
    )
    const views = await Promise.all(pendingViews)
    const allVariables = await client.variables.getAll(org.id)
    const variables = filterUnusedVars(allVariables, views)
    const exportedVariables = exportVariables(variables, allVariables)
    const dashboardTemplate = dashboardToTemplate(
      dashboard,
      views,
      exportedVariables
    )

    dispatch(setExportTemplate(RemoteDataState.Done, dashboardTemplate))
  } catch (error) {
    dispatch(setExportTemplate(RemoteDataState.Error))
    dispatch(notify(copy.createTemplateFailed(error)))
  }
}

export const saveVEOView = (dashboard: Dashboard) => async (
  dispatch,
  getState: GetState
): Promise<void> => {
  const view = getSaveableView(getState())

  try {
    if (view.id) {
      await dispatch(updateView(dashboard, view))
    } else {
      await dispatch(createCellWithView(dashboard, view))
    }
  } catch (error) {
    console.error(error)
    dispatch(notify(copy.cellAddFailed()))
  }
}
