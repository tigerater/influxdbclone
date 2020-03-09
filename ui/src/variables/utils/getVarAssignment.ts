// Utils
import {parseDuration} from 'src/shared/utils/duration'

// Types
import {VariableValues, VariableAssignment} from 'src/types'

export const getVarAssignment = (
  name: string,
  {selectedValue, valueType}: VariableValues
): VariableAssignment => {
  const assignment = {
    type: 'VariableAssignment' as 'VariableAssignment',
    id: {type: 'Identifier' as 'Identifier', name},
  }

  switch (valueType) {
    case 'boolean':
      return {
        ...assignment,
        init: {
          type: 'BooleanLiteral',
          value: selectedValue === 'true' ? true : false,
        },
      }
    case 'unsignedLong':
      return {
        ...assignment,
        init: {
          type: 'UnsignedIntegerLiteral',
          value: Number(selectedValue),
        },
      }
    case 'long':
      return {
        ...assignment,
        init: {
          type: 'IntegerLiteral',
          value: Number(selectedValue),
        },
      }
    case 'double':
      return {
        ...assignment,
        init: {
          type: 'FloatLiteral',
          value: Number(selectedValue),
        },
      }
    case 'string':
      return {
        ...assignment,
        init: {
          type: 'StringLiteral',
          value: selectedValue,
        },
      }
    case 'dateTime':
      return {
        ...assignment,
        init: {
          type: 'DateTimeLiteral',
          value: selectedValue,
        },
      }
    case 'duration':
      return {
        ...assignment,
        init: {
          type: 'DurationLiteral',
          values: parseDuration(selectedValue),
        },
      }
    default:
      throw new Error(
        `cannot form VariableAssignment from value of type "${valueType}"`
      )
  }
}
