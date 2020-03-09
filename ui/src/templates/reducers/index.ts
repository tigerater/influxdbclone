import {produce} from 'immer'
import {
  Action,
  ADD_TEMPLATE_SUMMARY,
  POPULATE_TEMPLATE_SUMMARIES,
  REMOVE_TEMPLATE_SUMMARY,
  SET_EXPORT_TEMPLATE,
  SET_TEMPLATE_SUMMARY,
  SET_TEMPLATES_STATUS,
} from 'src/templates/actions/creators'
import {
  ResourceType,
  RemoteDataState,
  TemplateSummary,
  TemplatesState,
} from 'src/types'
import {
  addResource,
  removeResource,
  setResource,
  setResourceAtID,
} from 'src/resources/reducers/helpers'

export const defaultState = (): TemplatesState => ({
  status: RemoteDataState.NotStarted,
  byID: {},
  allIDs: [],
  exportTemplate: {
    status: RemoteDataState.NotStarted,
    item: null,
  },
})

export const templatesReducer = (
  state: TemplatesState = defaultState(),
  action: Action
): TemplatesState =>
  produce(state, draftState => {
    switch (action.type) {
      case POPULATE_TEMPLATE_SUMMARIES: {
        setResource<TemplateSummary>(draftState, action, ResourceType.Templates)

        return
      }

      case SET_TEMPLATES_STATUS: {
        const {status} = action
        draftState.status = status
        return
      }

      case SET_TEMPLATE_SUMMARY: {
        setResourceAtID<TemplateSummary>(
          draftState,
          action,
          ResourceType.Templates
        )

        return
      }

      case SET_EXPORT_TEMPLATE: {
        const {status, item} = action
        draftState.exportTemplate.status = status

        if (item) {
          draftState.exportTemplate.item = item
        } else {
          draftState.exportTemplate.item = null
        }
        return
      }

      case REMOVE_TEMPLATE_SUMMARY: {
        removeResource<TemplateSummary>(draftState, action)

        return
      }

      case ADD_TEMPLATE_SUMMARY: {
        addResource<TemplateSummary>(draftState, action, ResourceType.Templates)

        return
      }
    }
  })

export default templatesReducer
