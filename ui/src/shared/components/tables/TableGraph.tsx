import React, {PureComponent} from 'react'
import _ from 'lodash'
import {ErrorHandling} from 'src/shared/decorators/errors'

import {
  ASCENDING,
  DESCENDING,
  DEFAULT_SORT_DIRECTION,
} from 'src/shared/constants/tableGraph'
import {TableViewProperties, SortOptions, FluxTable, TimeZone} from 'src/types'
import TableGraphTransform from 'src/shared/components/tables/TableGraphTransform'
import TableGraphTable from 'src/shared/components/tables/TableGraphTable'

interface Props {
  table: FluxTable
  properties: TableViewProperties
  timeZone: TimeZone
}

interface State {
  sortOptions: SortOptions
}

@ErrorHandling
class TableGraph extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)
    const sortField = _.get(
      props,
      'properties.tableOptions.sortBy.internalName'
    )

    this.state = {
      sortOptions: {
        field: sortField,
        direction: ASCENDING,
      },
    }
  }

  public render() {
    const {table, properties, timeZone} = this.props
    const {sortOptions} = this.state
    return (
      <TableGraphTransform
        data={table.data}
        properties={properties}
        dataTypes={table.dataTypes}
        sortOptions={sortOptions}
      >
        {transformedDataBundle => (
          <TableGraphTable
            properties={properties}
            dataTypes={table.dataTypes}
            onSort={this.handleSetSort}
            transformedDataBundle={transformedDataBundle}
            timeZone={timeZone}
          />
        )}
      </TableGraphTransform>
    )
  }

  public handleSetSort = (fieldName: string) => {
    this.setState(({sortOptions}) => {
      const newSortOptions = {...sortOptions}
      if (fieldName === sortOptions.field) {
        newSortOptions.direction =
          sortOptions.direction === ASCENDING ? DESCENDING : ASCENDING
      } else {
        newSortOptions.field = fieldName
        newSortOptions.direction = DEFAULT_SORT_DIRECTION
      }
      return {sortOptions: newSortOptions}
    })
  }
}

export default TableGraph
