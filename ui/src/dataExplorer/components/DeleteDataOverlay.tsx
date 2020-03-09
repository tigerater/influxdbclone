// Libraries
import React, {FunctionComponent} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'
import {Overlay} from '@influxdata/clockface'
import {get} from 'lodash'

// Components
import DeleteDataForm from 'src/shared/components/DeleteDataForm/DeleteDataForm'
import GetResources, {ResourceType} from 'src/shared/components/GetResources'

// Utils
import {getActiveQuery, getActiveTimeMachine} from 'src/timeMachine/selectors'

// Types
import {AppState, TimeRange} from 'src/types'

const resolveTimeRange = (timeRange: TimeRange): [number, number] | null => {
  const [lower, upper] = [
    Date.parse(timeRange.lower),
    Date.parse(timeRange.upper),
  ]

  if (!isNaN(lower) && !isNaN(upper)) {
    return [lower, upper]
  }

  return null
}

interface StateProps {
  selectedBucketName?: string
  selectedTimeRange?: [number, number]
}

const DeleteDataOverlay: FunctionComponent<StateProps & WithRouterProps> = ({
  router,
  params: {orgID},
  selectedBucketName,
  selectedTimeRange,
}) => {
  const handleDismiss = () => router.push(`/orgs/${orgID}/data-explorer`)

  return (
    <Overlay visible={true}>
      <Overlay.Container maxWidth={600}>
        <Overlay.Header title="Delete Data" onDismiss={handleDismiss} />
        <Overlay.Body>
          <GetResources resources={[ResourceType.Buckets]}>
            <DeleteDataForm
              handleDismiss={handleDismiss}
              initialBucketName={selectedBucketName}
              initialTimeRange={selectedTimeRange}
              orgID={orgID}
            />
          </GetResources>
        </Overlay.Body>
      </Overlay.Container>
    </Overlay>
  )
}

const mstp = (state: AppState): StateProps => {
  const activeQuery = getActiveQuery(state)
  const selectedBucketName = get(activeQuery, 'builderConfig.buckets.0')

  const {timeRange} = getActiveTimeMachine(state)
  const selectedTimeRange = resolveTimeRange(timeRange)

  return {
    selectedBucketName,
    selectedTimeRange,
  }
}

export default connect<StateProps>(mstp)(
  withRouter<StateProps>(DeleteDataOverlay)
)
