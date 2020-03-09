// Libraries
import React, {FC, createContext, useState} from 'react'
import {get} from 'lodash'

// Components
import {Plot} from '@influxdata/giraffe'
import CheckPlot from 'src/shared/components/CheckPlot'
import EmptyQueryView, {ErrorFormat} from 'src/shared/components/EmptyQueryView'

// Types
import {Check, TimeZone, CheckViewProperties} from 'src/types'
import TimeSeries from 'src/shared/components/TimeSeries'
import {createView} from 'src/shared/utils/view'
import {checkResultsLength} from 'src/shared/utils/vis'
import {getTimeRangeVars} from 'src/variables/utils/getTimeRangeVars'
import {TimeRange} from 'src/types'

interface ResourceIDs {
  checkIDs: {[x: string]: boolean}
  endpointIDs: {[x: string]: boolean}
  ruleIDs: {[x: string]: boolean}
}

export const ResourceIDsContext = createContext<ResourceIDs>(null)

interface OwnProps {
  check: Check
  timeZone: TimeZone
}

type Props = OwnProps
//TODO maybe update submitToken when we know how

const CheckHistoryVisualization: FC<Props> = ({check, timeZone}) => {
  const view = createView<CheckViewProperties>(get(check, 'type', 'threshold'))

  const [submitToken] = useState(0)
  const [manualRefresh] = useState(0)

  return (
    <TimeSeries
      submitToken={submitToken}
      queries={[check.query]}
      key={manualRefresh}
      variables={getTimeRangeVars({lower: 'now() - 5m'} as TimeRange)}
      check={check}
    >
      {({giraffeResult, loading, errorMessage, isInitialFetch, statuses}) => {
        return (
          <EmptyQueryView
            errorFormat={ErrorFormat.Tooltip}
            errorMessage={errorMessage}
            hasResults={checkResultsLength(giraffeResult)}
            loading={loading}
            isInitialFetch={isInitialFetch}
            queries={[check.query]}
            fallbackNote={null}
          >
            <CheckPlot
              check={check}
              table={get(giraffeResult, 'table')}
              fluxGroupKeyUnion={get(giraffeResult, 'fluxGroupKeyUnion')}
              loading={loading}
              timeZone={timeZone}
              viewProperties={view.properties}
              statuses={statuses}
            >
              {config => <Plot config={config} />}
            </CheckPlot>
            {/* {<CheckHistoryStatuses .../>} */}
          </EmptyQueryView>
        )
      }}
    </TimeSeries>
  )
}

export default CheckHistoryVisualization
