// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'

// Components
import TimeMachine from 'src/timeMachine/components/TimeMachine'
import LimitChecker from 'src/cloud/components/LimitChecker'
import AssetLimitAlert from 'src/cloud/components/AssetLimitAlert'

// Actions
import {setActiveTimeMachine} from 'src/timeMachine/actions'

// Utils
import {DE_TIME_MACHINE_ID} from 'src/timeMachine/constants'
import {HoverTimeProvider} from 'src/dashboards/utils/hoverTime'
import {queryBuilderFetcher} from 'src/timeMachine/apis/QueryBuilderFetcher'
import {
  extractRateLimitResourceName,
  extractRateLimitStatus,
} from 'src/cloud/utils/limits'

// Types
import {AppState} from 'src/types'
import {LimitStatus} from 'src/cloud/actions/limits'

interface StateProps {
  resourceName: string
  limitStatus: LimitStatus
}

interface DispatchProps {
  onSetActiveTimeMachine: typeof setActiveTimeMachine
}

type Props = DispatchProps & StateProps
class DataExplorer extends PureComponent<Props, {}> {
  constructor(props: Props) {
    super(props)

    props.onSetActiveTimeMachine(DE_TIME_MACHINE_ID)
    queryBuilderFetcher.clearCache()
  }

  public render() {
    const {resourceName, limitStatus} = this.props

    return (
      <div className="data-explorer">
        <LimitChecker>
          <AssetLimitAlert
            resourceName={resourceName}
            limitStatus={limitStatus}
          >
            <HoverTimeProvider>
              <TimeMachine />
            </HoverTimeProvider>
          </AssetLimitAlert>
        </LimitChecker>
      </div>
    )
  }
}

const mstp = (state: AppState): StateProps => {
  const {
    cloud: {limits},
  } = state

  return {
    resourceName: extractRateLimitResourceName(limits),
    limitStatus: extractRateLimitStatus(limits),
  }
}

const mdtp: DispatchProps = {
  onSetActiveTimeMachine: setActiveTimeMachine,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(DataExplorer)
