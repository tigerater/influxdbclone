// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import {Link} from 'react-router'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import TemplatedCodeSnippet, {
  transform,
} from 'src/shared/components/TemplatedCodeSnippet'
import BucketDropdown from 'src/dataLoaders/components/BucketsDropdown'
import {
  ComponentColor,
  Button,
  // SpinnerContainer,
  //  TechnoSpinner,
  Overlay,
} from '@influxdata/clockface'

// Utils
import {downloadTextFile} from 'src/shared/utils/download'

// Types
import {AppState, Bucket} from 'src/types'

interface OwnProps {
  onClose: () => void
}

interface StateProps {
  org: string
  orgID: string
  server: string
  buckets: Bucket[]
}

type Props = OwnProps & StateProps

const TELEGRAF_OUTPUT = ` [[outputs.influxdb_v2]]
  ## The URLs of the InfluxDB cluster nodes.
  ##
  ## Multiple URLs can be specified for a single cluster, only ONE of the
  ## urls will be written to each interval.
  ## urls exp: http://127.0.0.1:9999
  urls = ["<%= server %>"]

  ## Token for authentication.
  token = "<%= token %>"

  ## Organization is the name of the organization you wish to write to; must exist.
  organization = "<%= org %>"

  ## Destination bucket to write into.
  bucket = "<%= bucket %>"
`

const OUTPUT_DEFAULTS = {
  server: 'http://127.0.0.1:9999',
  token: '$INFLUX_TOKEN',
  org: 'orgID',
  bucket: 'bucketID',
}

@ErrorHandling
class TelegrafOutputOverlay extends PureComponent<Props> {
  state = {
    selectedBucket: null,
  }

  public render() {
    return <>{this.overlay}</>
  }

  private get overlay(): JSX.Element {
    const {server, org, orgID, buckets} = this.props
    const _buckets = (buckets || [])
      .filter(item => item.type !== 'system')
      .sort((a, b) => {
        const _a = a.name.toLowerCase()
        const _b = b.name.toLowerCase()
        return _a > _b ? 1 : _a < _b ? -1 : 0
      })
    const {selectedBucket} = this.state
    let bucket_dd = null
    let bucket = null

    if (_buckets.length) {
      bucket = selectedBucket ? selectedBucket : _buckets[0]
      bucket_dd = (
        <BucketDropdown
          selectedBucketID={bucket.id}
          buckets={_buckets}
          onSelectBucket={this.handleSelectBucket}
        />
      )
    }

    return (
      <Overlay.Container maxWidth={1200}>
        <Overlay.Header
          title="Telegraf Output Configuration"
          onDismiss={this.handleDismiss}
        />
        <Overlay.Body>
          <p style={{marginTop: '-18px'}}>
            The $INFLUX_TOKEN can be generated on the
            <Link to={`/orgs/${orgID}/load-data/tokens`}>&nbsp;Tokens tab</Link>
            .
          </p>
          {bucket_dd}
          <div className="output-overlay">
            <TemplatedCodeSnippet
              template={TELEGRAF_OUTPUT}
              label="telegraf.conf"
              testID="telegraf-output-overlay--code-snippet"
              values={{
                server,
                org,
                bucket: bucket ? bucket.name : OUTPUT_DEFAULTS.bucket,
              }}
              defaults={OUTPUT_DEFAULTS}
            />
          </div>
        </Overlay.Body>
        <Overlay.Footer>
          <Button
            color={ComponentColor.Secondary}
            text="Download Config"
            onClick={this.handleDownloadConfig}
          />
        </Overlay.Footer>
      </Overlay.Container>
    )
  }

  private handleDismiss = () => {
    this.props.onClose()
  }

  private handleSelectBucket = bucket => {
    this.setState({
      selectedBucket: bucket,
    })
  }

  private handleDownloadConfig = () => {
    const config = transform(
      TELEGRAF_OUTPUT,
      Object.assign({}, OUTPUT_DEFAULTS, {})
    )
    downloadTextFile(config, 'outputs.influxdb_v2', '.conf')
  }
}

const mstp = (state: AppState): StateProps => {
  const org = state.orgs.org.name
  const orgID = state.orgs.org.id
  const server = window.location.origin

  return {
    org,
    orgID,
    server,
    buckets: state.buckets.list,
  }
}

export {TelegrafOutputOverlay}
export default connect<StateProps, {}, {}>(
  mstp,
  null
)(TelegrafOutputOverlay)
