// Libraries
import React, {FunctionComponent} from 'react'
import {Button, IconFont, ButtonShape, Panel} from '@influxdata/clockface'

// Components
import FilterRow from 'src/shared/components/DeleteDataForm/FilterRow'

export interface Filter {
  key: string
  value: string
}

interface Props {
  filters: Filter[]
  onSetFilter: (filter: Filter, index: number) => any
  onDeleteFilter: (index: number) => any
  shouldValidate: boolean
}

const FilterEditor: FunctionComponent<Props> = ({
  filters,
  onSetFilter,
  onDeleteFilter,
  shouldValidate,
}) => {
  return (
    <div className="delete-data-filters">
      <Button
        text="Add Filter"
        icon={IconFont.Plus}
        shape={ButtonShape.StretchToFit}
        className="delete-data-filters--new-filter"
        onClick={() => onSetFilter({key: '', value: ''}, filters.length)}
      />
      {filters.length > 0 ? (
        <div className="delete-data-filters--filters">
          {filters.map((filter, i) => (
            <FilterRow
              key={i}
              filter={filter}
              onChange={filter => onSetFilter(filter, i)}
              onDelete={() => onDeleteFilter(i)}
              shouldValidate={shouldValidate}
            />
          ))}
        </div>
      ) : (
        <Panel className="delete-data-filters--no-filters">
          <Panel.Body>
            <p>
              If no filters are specified, all data points in the selected time
              range will be marked for deletion.
            </p>
          </Panel.Body>
        </Panel>
      )}
    </div>
  )
}

export default FilterEditor
