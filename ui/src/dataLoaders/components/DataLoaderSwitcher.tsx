// Libraries
import React, {PureComponent} from 'react'
import _ from 'lodash'

// Components
import DataLoadersWizard from 'src/dataLoaders/components/DataLoadersWizard'
import CollectorsWizard from 'src/dataLoaders/components/collectorsWizard/CollectorsWizard'
import LineProtocolWizard from 'src/dataLoaders/components/lineProtocolWizard/LineProtocolWizard'

// Types
import {Substep, DataLoaderType} from 'src/types/v2/dataLoaders'
import {Bucket} from '@influxdata/influx'

interface Props {
  type: DataLoaderType
  onCompleteSetup: () => void
  visible: boolean
  buckets: Bucket[]
  startingType?: DataLoaderType
  startingStep?: number
  startingSubstep?: Substep
}

class DataLoaderSwitcher extends PureComponent<Props> {
  public render() {
    const {
      buckets,
      type,
      visible,
      onCompleteSetup,
      startingStep,
      startingSubstep,
      startingType,
    } = this.props

    switch (type) {
      case DataLoaderType.Scraping:
      case DataLoaderType.Empty:
        return (
          <DataLoadersWizard
            visible={visible}
            onCompleteSetup={onCompleteSetup}
            buckets={buckets}
            startingStep={startingStep}
            startingSubstep={startingSubstep}
            startingType={startingType}
          />
        )
      case DataLoaderType.Streaming:
        return (
          <CollectorsWizard
            visible={visible}
            onCompleteSetup={onCompleteSetup}
            buckets={buckets}
          />
        )
      case DataLoaderType.LineProtocol:
        return (
          <LineProtocolWizard
            onCompleteSetup={onCompleteSetup}
            visible={visible}
            buckets={buckets}
          />
        )
    }
  }
}

export default DataLoaderSwitcher
