// Libraries
import React, {FC, useEffect} from 'react'
import moment from 'moment'
import {connect} from 'react-redux'
import {Form, Grid, Columns, Panel} from '@influxdata/clockface'

// Components
import BucketsDropdown from 'src/shared/components/DeleteDataForm/BucketsDropdown'
import TimeRangeDropdown from 'src/shared/components/DeleteDataForm/TimeRangeDropdown'
import Checkbox from 'src/shared/components/Checkbox'
import DeleteButton from 'src/shared/components/DeleteDataForm/DeleteButton'
import FilterEditor from 'src/shared/components/DeleteDataForm/FilterEditor'

// Types
import {Filter, RemoteDataState} from 'src/types'

// Selectors
import {setCanDelete} from 'src/shared/selectors/canDelete'

// Actions
import {
  deleteFilter,
  deleteWithPredicate,
  resetFilters,
  setDeletionStatus,
  setFilter,
  setIsSerious,
  setBucketAndKeys,
  setTimeRange,
} from 'src/shared/actions/predicates'

interface OwnProps {
  orgID: string
  handleDismiss: () => void
  initialBucketName?: string
  initialTimeRange?: [number, number]
  keys: string[]
  values: (string | number)[]
}

interface StateProps {
  bucketName: string
  canDelete: boolean
  deletionStatus: RemoteDataState
  filters: Filter[]
  isSerious: boolean
  keys: string[]
  timeRange: [number, number]
  values: (string | number)[]
}

interface DispatchProps {
  deleteFilter: (index: number) => void
  deleteWithPredicate: typeof deleteWithPredicate
  resetFilters: () => void
  setDeletionStatus: (status: RemoteDataState) => void
  setFilter: typeof setFilter
  setIsSerious: (isSerious: boolean) => void
  setBucketAndKeys: (orgID: string, bucketName: string) => void
  setTimeRange: (timeRange: [number, number]) => void
}

export type Props = StateProps & DispatchProps & OwnProps

const DeleteDataForm: FC<Props> = ({
  bucketName,
  canDelete,
  deleteFilter,
  deletionStatus,
  deleteWithPredicate,
  filters,
  handleDismiss,
  initialBucketName,
  initialTimeRange,
  isSerious,
  keys,
  orgID,
  resetFilters,
  setDeletionStatus,
  setFilter,
  setIsSerious,
  setBucketAndKeys,
  setTimeRange,
  timeRange,
  values,
}) => {
  const name = bucketName || initialBucketName
  // trigger the setBucketAndKeys if the bucketName hasn't been set
  if (bucketName === '' && name !== undefined) {
    useEffect(() => {
      setBucketAndKeys(orgID, name)
    })
  }

  const realTimeRange = initialTimeRange || timeRange

  const formatPredicates = predicates => {
    const result = []
    predicates.forEach(predicate => {
      const {key, equality, value} = predicate
      result.push(`${key} ${equality} ${value}`)
    })
    return result.join(' AND ')
  }

  const handleDelete = () => {
    setDeletionStatus(RemoteDataState.Loading)

    const [start, stop] = realTimeRange

    const data = {
      start: moment(start).toISOString(),
      stop: moment(stop).toISOString(),
    }

    if (filters.length > 0) {
      data['predicate'] = formatPredicates(filters)
    }

    const params = {
      data,
      query: {
        orgID,
        bucket: name,
      },
    }

    deleteWithPredicate(params)
    handleDismiss()
  }

  const handleBucketClick = selectedBucket => {
    setBucketAndKeys(orgID, selectedBucket)
    resetFilters()
  }

  return (
    <Form className="delete-data-form">
      <Grid>
        <Grid.Row>
          <Grid.Column widthXS={Columns.Four}>
            <Form.Element label="Target Bucket">
              <BucketsDropdown
                bucketName={name}
                onSetBucketName={bucketName => handleBucketClick(bucketName)}
              />
            </Form.Element>
          </Grid.Column>
          <Grid.Column widthXS={Columns.Eight}>
            <Form.Element label="Time Range">
              <TimeRangeDropdown
                timeRange={realTimeRange}
                onSetTimeRange={timeRange => setTimeRange(timeRange)}
              />
            </Form.Element>
          </Grid.Column>
        </Grid.Row>
        <Grid.Row>
          <Grid.Column widthXS={Columns.Twelve}>
            <FilterEditor
              bucket={name}
              filters={filters}
              keys={keys}
              onDeleteFilter={index => deleteFilter(index)}
              onSetFilter={(filter, index) => setFilter(filter, index)}
              orgID={orgID}
              shouldValidate={isSerious}
              values={values}
            />
          </Grid.Column>
        </Grid.Row>
        <Grid.Row>
          <Grid.Column widthXS={Columns.Twelve}>
            <Panel className="delete-data-form--danger-zone">
              <Panel.Header>
                <Panel.Title>Danger Zone!</Panel.Title>
              </Panel.Header>
              <Panel.Body className="delete-data-form--confirm">
                <Checkbox
                  testID="delete-checkbox"
                  label="I understand that this cannot be undone."
                  checked={isSerious}
                  onSetChecked={isSerious => setIsSerious(isSerious)}
                />
                <DeleteButton
                  status={deletionStatus}
                  valid={canDelete}
                  onClick={handleDelete}
                />
              </Panel.Body>
            </Panel>
          </Grid.Column>
        </Grid.Row>
      </Grid>
    </Form>
  )
}

const mstp = ({predicates}) => {
  const {
    bucketName,
    deletionStatus,
    filters,
    isSerious,
    keys,
    timeRange,
    values,
  } = predicates
  return {
    bucketName,
    canDelete: setCanDelete(predicates),
    deletionStatus,
    filters,
    isSerious,
    keys,
    timeRange,
    values,
  }
}

const mdtp = {
  deleteFilter,
  deleteWithPredicate,
  resetFilters,
  setDeletionStatus,
  setFilter,
  setIsSerious,
  setBucketAndKeys,
  setTimeRange,
}

export default connect<StateProps, DispatchProps>(
  mstp,
  mdtp
)(DeleteDataForm)
