import {produce} from 'immer'
import {Actions, ActionTypes} from 'src/templates/actions/'
import {TemplateSummary, DocumentCreate} from '@influxdata/influx'
import {RemoteDataState} from 'src/types'

export interface TemplatesState {
  status: RemoteDataState
  items: TemplateSummary[]
  exportTemplate: {status: RemoteDataState; item: DocumentCreate}
}

export const defaultState = (): TemplatesState => ({
  status: RemoteDataState.NotStarted,
  items: [],
  exportTemplate: {
    status: RemoteDataState.NotStarted,
    item: null,
  },
})

export const templatesReducer = (
  state: TemplatesState = defaultState(),
  action: Actions
): TemplatesState =>
  produce(state, draftState => {
    switch (action.type) {
      case ActionTypes.PopulateTemplateSummaries: {
        const {status, items} = action.payload
        draftState.status = status
        if (items) {
          draftState.items = items
        } else {
          draftState.items = null
        }
        return
      }

      case ActionTypes.SetTemplatesStatus: {
        const {status} = action.payload
        draftState.status = status
        return
      }

      case ActionTypes.SetTemplateSummary: {
        const updated = draftState.items.map(t => {
          if (t.id === action.payload.id) {
            return action.payload.templateSummary
          }

          return t
        })

        draftState.items = updated

        return
      }

      case ActionTypes.SetExportTemplate: {
        const {status, item} = action.payload
        draftState.exportTemplate.status = status

        if (item) {
          draftState.exportTemplate.item = item
        } else {
          draftState.exportTemplate.item = null
        }
        return
      }

      case ActionTypes.RemoveTemplateSummary: {
        const {templateID} = action.payload
        const {items} = draftState
        draftState.items = items.filter(l => {
          return l.id !== templateID
        })

        return
      }

      case ActionTypes.AddTemplateSummary: {
        const {item} = action.payload
        const {items} = draftState

        draftState.items = [...items, item]

        return
      }
    }
  })

export default templatesReducer
