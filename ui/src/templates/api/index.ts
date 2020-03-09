import _, {get} from 'lodash'
import {
  DashboardTemplate,
  TemplateType,
  CellIncluded,
  LabelIncluded,
  ViewIncluded,
  TaskTemplate,
  TemplateBase,
  Task,
  VariableTemplate,
  Variable,
} from 'src/types'
import {IDashboard, Cell} from '@influxdata/influx'
import {client} from 'src/utils/api'

// Utils
import {
  findIncludedsFromRelationships,
  findLabelsToCreate,
  findIncludedFromRelationship,
  findVariablesToCreate,
  findIncludedVariables,
  hasLabelsRelationships,
  getLabelRelationships,
} from 'src/templates/utils/'
import {addDefaults} from 'src/tasks/actions'
import {addVariableDefaults} from 'src/variables/actions'
import {addLabelDefaults} from 'src/labels/utils'
// API
import {
  getTask as apiGetTask,
  postTask as apiPostTask,
  postTasksLabel as apiPostTasksLabel,
  getLabels as apiGetLabels,
  postLabel as apiPostLabel,
  getVariable as apiGetVariable,
  getVariables as apiGetVariables,
  postVariable as apiPostVariable,
  postVariablesLabel as apiPostVariablesLabel,
} from 'src/client'
// Create Dashboard Templates

export const createDashboardFromTemplate = async (
  template: DashboardTemplate,
  orgID: string
): Promise<IDashboard> => {
  const {content} = template

  if (
    content.data.type !== TemplateType.Dashboard ||
    template.meta.version !== '1'
  ) {
    throw new Error('Cannot create dashboard from this template')
  }

  const createdDashboard = await client.dashboards.create({
    ...content.data.attributes,
    orgID,
  })

  if (!createdDashboard || !createdDashboard.id) {
    throw new Error('Failed to create dashboard from template')
  }

  // associate imported label id with new label
  const labelMap = await createLabelsFromTemplate(template, orgID)

  await Promise.all([
    await addDashboardLabelsFromTemplate(template, labelMap, createdDashboard),
    await createCellsFromTemplate(template, createdDashboard),
  ])

  await createVariablesFromTemplate(template, labelMap, orgID)

  const dashboard = await client.dashboards.get(createdDashboard.id)
  return dashboard
}

const addDashboardLabelsFromTemplate = async (
  template: DashboardTemplate,
  labelMap: LabelMap,
  dashboard: IDashboard
) => {
  const labelRelationships = getLabelRelationships(template.content.data)
  const labelIDs = labelRelationships.map(l => labelMap[l.id] || '')

  await client.dashboards.addLabels(dashboard.id, labelIDs)
}

type LabelMap = {[importedID: string]: CreatedLabelID}
type CreatedLabelID = string

const createLabelsFromTemplate = async <T extends TemplateBase>(
  template: T,
  orgID: string
): Promise<LabelMap> => {
  const {
    content: {data, included},
  } = template

  const labeledResources = [data, ...included].filter(r =>
    hasLabelsRelationships(r)
  )

  if (_.isEmpty(labeledResources)) {
    return {}
  }

  const labelRelationships = _.flatMap(labeledResources, r =>
    getLabelRelationships(r)
  )

  const includedLabels = findIncludedsFromRelationships<LabelIncluded>(
    included,
    labelRelationships
  )

  const resp = await apiGetLabels({query: {orgID}})

  if (resp.status !== 200) {
    throw new Error(resp.data.message)
  }

  const existingLabels = resp.data.labels.map(l => addLabelDefaults(l))

  const foundLabelsToCreate = findLabelsToCreate(
    existingLabels,
    includedLabels
  ).map(l => ({
    orgID,
    name: _.get(l, 'attributes.name', ''),
    properties: _.get(l, 'attributes.properties', {}),
  }))

  const promisedLabels = foundLabelsToCreate.map(async lab => {
    return apiPostLabel({
      data: lab,
    })
      .then(res => get(res, 'res.data.label', ''))
      .then(lab => addLabelDefaults(lab))
  })

  const createdLabels = await Promise.all(promisedLabels)

  const allLabels = [...createdLabels, ...existingLabels]

  const labelMap: LabelMap = {}

  includedLabels.forEach(label => {
    const createdLabel = allLabels.find(l => l.name === label.attributes.name)

    labelMap[label.id] = createdLabel.id
  })

  return labelMap
}

const createCellsFromTemplate = async (
  template: DashboardTemplate,
  createdDashboard: IDashboard
) => {
  const {
    content: {data, included},
  } = template

  if (!data.relationships || !data.relationships[TemplateType.Cell]) {
    return
  }

  const cellRelationships = data.relationships[TemplateType.Cell].data

  const cellsToCreate = findIncludedsFromRelationships<CellIncluded>(
    included,
    cellRelationships
  )

  const pendingCells = cellsToCreate.map(c => {
    const {
      attributes: {x, y, w, h},
    } = c
    return client.dashboards.createCell(createdDashboard.id, {x, y, w, h})
  })

  const cellResponses = await Promise.all(pendingCells)

  createViewsFromTemplate(
    template,
    cellResponses,
    cellsToCreate,
    createdDashboard.id
  )
}

const createViewsFromTemplate = async (
  template: DashboardTemplate,
  cellResponses: Cell[],
  cellsToCreate: CellIncluded[],
  dashboardID: string
) => {
  const viewsToCreate = cellsToCreate.map(c => {
    const {
      content: {included},
    } = template

    const viewRelationship = c.relationships[TemplateType.View].data

    return findIncludedFromRelationship<ViewIncluded>(
      included,
      viewRelationship
    )
  })

  const pendingViews = viewsToCreate.map((v, i) => {
    return client.dashboards.updateView(
      dashboardID,
      cellResponses[i].id,
      v.attributes
    )
  })

  await Promise.all(pendingViews)
}

const createVariablesFromTemplate = async (
  template: DashboardTemplate | VariableTemplate,
  labelMap: LabelMap,
  orgID: string
) => {
  const {
    content: {data, included},
  } = template
  if (!data.relationships || !data.relationships[TemplateType.Variable]) {
    return
  }
  const variablesIncluded = findIncludedVariables(included)

  const resp = await apiGetVariables({query: {orgID}})
  if (resp.status !== 200) {
    throw new Error(resp.data.message)
  }

  const variables = resp.data.variables.map(v => addVariableDefaults(v))

  const variablesToCreate = findVariablesToCreate(
    variables,
    variablesIncluded
  ).map(v => ({...v.attributes, orgID}))

  const pendingVariables = variablesToCreate.map(vars =>
    apiPostVariable({data: vars})
  )

  const resolvedVariables = await Promise.all(pendingVariables)
  if (
    resolvedVariables.length > 0 &&
    resolvedVariables.every(r => r.status !== 201)
  ) {
    throw new Error('An error occurred creating the variables from templates')
  }

  const createdVariables = await Promise.all(pendingVariables).then(vars =>
    vars.map(res => addVariableDefaults(res.data as Variable))
  )

  const allVars = [...variables, ...createdVariables]

  const addLabelsToVars = variablesIncluded.map(async includedVar => {
    const variable = allVars.find(v => v.name === includedVar.attributes.name)
    const labelRelationships = getLabelRelationships(includedVar)
    const labelIDs = labelRelationships.map(l => labelMap[l.id] || '')
    const pending = labelIDs.map(async labelID => {
      await apiPostVariablesLabel({variableID: variable.id, data: {labelID}})
    })
    await Promise.all(pending)
  })

  await Promise.all(addLabelsToVars)
}

export const createTaskFromTemplate = async (
  template: TaskTemplate,
  orgID: string
): Promise<Task> => {
  const {content} = template
  try {
    if (
      content.data.type !== TemplateType.Task ||
      template.meta.version !== '1'
    ) {
      throw new Error('Cannot create task from this template')
    }

    const flux = content.data.attributes.flux

    const postResp = await apiPostTask({data: {orgID, flux}})

    if (postResp.status !== 201) {
      throw new Error(postResp.data.message)
    }

    const postedTask = addDefaults(postResp.data)

    // associate imported label.id with created label
    const labelMap = await createLabelsFromTemplate(template, orgID)

    await addTaskLabelsFromTemplate(template, labelMap, postedTask)

    const resp = await apiGetTask({taskID: postedTask.id})

    if (resp.status !== 200) {
      throw new Error(resp.data.message)
    }

    const task = addDefaults(resp.data)

    return task
  } catch (e) {
    console.error(e)
  }
}

const addTaskLabelsFromTemplate = async (
  template: TaskTemplate,
  labelMap: LabelMap,
  task: Task
) => {
  try {
    const relationships = getLabelRelationships(template.content.data)
    const labelIDs = relationships.map(l => labelMap[l.id] || '')
    const pending = labelIDs.map(labelID =>
      apiPostTasksLabel({taskID: task.id, data: {labelID}})
    )
    const resolved = await Promise.all(pending)
    if (resolved.length > 0 && resolved.some(r => r.status !== 201)) {
      throw new Error('An error occurred adding task labels from the templates')
    }
  } catch (e) {
    console.error(e)
  }
}

export const createVariableFromTemplate = async (
  template: VariableTemplate,
  orgID: string
) => {
  const {content} = template
  try {
    if (
      content.data.type !== TemplateType.Variable ||
      template.meta.version !== '1'
    ) {
      throw new Error('Cannot create variable from this template')
    }

    const resp = await apiPostVariable({
      data: {
        ...content.data.attributes,
        orgID,
      },
    })

    if (resp.status !== 201) {
      throw new Error(resp.data.message)
    }

    // associate imported label.id with created label
    const labelsMap = await createLabelsFromTemplate(template, orgID)

    await createVariablesFromTemplate(template, labelsMap, orgID)

    const variable = await apiGetVariable({variableID: resp.data.id})

    if (variable.status !== 200) {
      throw new Error(variable.data.message)
    }

    return addVariableDefaults(variable.data)
  } catch (e) {
    console.error(e)
  }
}
