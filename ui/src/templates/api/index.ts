import _ from 'lodash'
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
} from 'src/types'
import {IDashboard, Cell} from '@influxdata/influx'
import {client} from 'src/utils/api'

import {
  findIncludedsFromRelationships,
  findLabelsToCreate,
  findIncludedFromRelationship,
  findVariablesToCreate,
  findIncludedVariables,
  hasLabelsRelationships,
  getLabelRelationships,
} from 'src/templates/utils/'

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
    throw new Error('Can not create dashboard from this template')
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

  const existingLabels = await client.labels.getAll(orgID)

  const labelsToCreate = findLabelsToCreate(existingLabels, includedLabels).map(
    l => ({
      orgID,
      name: _.get(l, 'attributes.name', ''),
      properties: _.get(l, 'attributes.properties', {}),
    })
  )

  const createdLabels = await client.labels.createAll(labelsToCreate)
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

  const existingVariables = await client.variables.getAll(orgID)

  const variablesToCreate = findVariablesToCreate(
    existingVariables,
    variablesIncluded
  ).map(v => ({...v.attributes, orgID}))

  const createdVariables = await client.variables.createAll(variablesToCreate)

  const allVars = [...existingVariables, ...createdVariables]

  const addLabelsToVars = variablesIncluded.map(async includedVar => {
    const variable = allVars.find(v => v.name === includedVar.attributes.name)
    const labelRelationships = getLabelRelationships(includedVar)
    const labelIDs = labelRelationships.map(l => labelMap[l.id] || '')

    await client.variables.addLabels(variable.id, labelIDs)
  })

  await Promise.all(addLabelsToVars)
}

export const createTaskFromTemplate = async (
  template: TaskTemplate,
  orgID: string
): Promise<Task> => {
  const {content} = template

  if (
    content.data.type !== TemplateType.Task ||
    template.meta.version !== '1'
  ) {
    throw new Error('Can not create task from this template')
  }

  const flux = content.data.attributes.flux

  const createdTask = await client.tasks.createByOrgID(orgID, flux, null)

  if (!createdTask || !createdTask.id) {
    throw new Error('Could not create task')
  }

  // associate imported label.id with created label
  const labelMap = await createLabelsFromTemplate(template, orgID)

  await addTaskLabelsFromTemplate(template, labelMap, createdTask)

  const task = await client.tasks.get(createdTask.id)

  return task
}

const addTaskLabelsFromTemplate = async (
  template: TaskTemplate,
  labelMap: LabelMap,
  task: Task
) => {
  const relationships = getLabelRelationships(template.content.data)
  const labelIDs = relationships.map(l => labelMap[l.id] || '')
  await client.tasks.addLabels(task.id, labelIDs)
}

export const createVariableFromTemplate = async (
  template: VariableTemplate,
  orgID: string
) => {
  const {content} = template

  if (
    content.data.type !== TemplateType.Variable ||
    template.meta.version !== '1'
  ) {
    throw new Error('Can not create variable from this template')
  }

  const createdVariable = await client.variables.create({
    ...content.data.attributes,
    orgID,
  })

  if (!createdVariable || !createdVariable.id) {
    throw new Error('Failed to create variable from template')
  }

  // associate imported label.id with created label
  const labelsMap = await createLabelsFromTemplate(template, orgID)

  await createVariablesFromTemplate(template, labelsMap, orgID)

  const variable = await client.variables.get(createdVariable.id)

  return variable
}
