// Libraries
import {produce} from 'immer'
import {get} from 'lodash'

// Types
import {
  RemoteDataState,
  VariableArgumentType,
  QueryArguments,
  MapArguments,
  CSVArguments,
  Variable,
} from 'src/types'
import {VariableValuesByID} from 'src/variables/types'
import {Action, EditorAction} from 'src/variables/actions'

export const initialEditorState = (): VariableEditorState => ({
  name: '',
  selected: 'query',
  argsQuery: null,
  argsMap: null,
  argsConstant: null,
})

export interface VariableEditorState {
  name: string
  selected: VariableArgumentType
  argsQuery: QueryArguments
  argsMap: MapArguments
  argsConstant: CSVArguments
}

export const variableEditorReducer = (
  state: VariableEditorState = initialEditorState(),
  action: EditorAction
): VariableEditorState =>
  produce(state, draftState => {
    switch (action.type) {
      case 'CLEAR_VARIABLE_EDITOR': {
        return initialEditorState()
      }
      case 'CHANGE_VARIABLE_EDITOR_TYPE': {
        draftState.selected = action.payload
        return
      }
      case 'UPDATE_VARIABLE_EDITOR_NAME': {
        draftState.name = action.payload
        return
      }
      case 'UPDATE_VARIABLE_EDITOR_QUERY': {
        draftState.argsQuery = action.payload
        return
      }
      case 'UPDATE_VARIABLE_EDITOR_MAP': {
        draftState.argsMap = action.payload
        return
      }
      case 'UPDATE_VARIABLE_EDITOR_CONSTANT': {
        draftState.argsConstant = action.payload
        return
      }
      default:
        return
    }
  })

export const initialState = (): VariablesState => ({
  status: RemoteDataState.NotStarted,
  variables: {},
  values: {},
})

export interface VariablesState {
  status: RemoteDataState // Loading status of the entire variables collection
  variables: {
    [variableID: string]: {
      status: RemoteDataState // Loading status of an individual variable
      variable: Variable
    }
  }
  values: {
    // Different variable values can be selected in different
    // "contexts"---different parts of the app like a particular dashboard, or
    // the Data Explorer
    [contextID: string]: {
      status: RemoteDataState
      order: string[] // IDs of variables
      values: VariableValuesByID
    }
  }
}

export const variablesReducer = (
  state: VariablesState = initialState(),
  action: Action
): VariablesState =>
  produce(state, draftState => {
    switch (action.type) {
      case 'SET_VARIABLES': {
        const {status, variables} = action.payload

        draftState.status = status

        if (variables) {
          draftState.variables = {}

          for (const variable of variables) {
            draftState.variables[variable.id] = {
              variable,
              status: RemoteDataState.Done,
            }
          }
        }

        return
      }

      case 'SET_VARIABLE': {
        const {id, status, variable} = action.payload
        const variableExists = !!draftState.variables[id]

        if (variable || !variableExists) {
          draftState.variables[id] = {variable, status}
        } else {
          draftState.variables[id].status = status
        }

        return
      }

      case 'REMOVE_VARIABLE': {
        const {id} = action.payload

        delete draftState.variables[id]

        return
      }

      case 'SET_VARIABLE_VALUES': {
        const {contextID, status, values} = action.payload
        const prevOrder = get(draftState, `values.${contextID}.order`, [])

        if (values) {
          const order = Object.keys(values).sort(
            (a, b) => prevOrder.indexOf(a) - prevOrder.indexOf(b)
          )

          draftState.values[contextID] = {
            status,
            values,
            order,
          }
        } else if (draftState.values[contextID]) {
          draftState.values[contextID].status = status
        } else {
          draftState.values[contextID] = {status, values: null, order: []}
        }

        return
      }

      case 'SELECT_VARIABLE_VALUE': {
        const {contextID, variableID, selectedValue} = action.payload

        const valuesExist = !!get(
          draftState,
          `values.${contextID}.values.${variableID}`
        )

        if (!valuesExist) {
          return
        }

        draftState.values[contextID].values[
          variableID
        ].selectedValue = selectedValue

        return
      }

      case 'MOVE_VARIABLE': {
        const {originalIndex, newIndex, contextID} = action.payload

        const variableIDToMove = get(
          draftState,
          `values.${contextID}.order[${originalIndex}]`
        )

        const variableIDToSwap = get(
          draftState,
          `values.${contextID}.order[${newIndex}]`
        )

        draftState.values[contextID].order[originalIndex] = variableIDToSwap
        draftState.values[contextID].order[newIndex] = variableIDToMove

        return
      }
    }
  })
