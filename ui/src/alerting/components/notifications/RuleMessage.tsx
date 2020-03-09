// Libraries
import React, {FC, useEffect} from 'react'
import {connect} from 'react-redux'

// Components
import {Form, Panel, Grid, Columns} from '@influxdata/clockface'
import RuleEndpointDropdown from 'src/alerting/components/notifications/RuleEndpointDropdown'
import RuleMessageContents from 'src/alerting/components/notifications/RuleMessageContents'

// Utils
import {getRuleVariantDefaults} from 'src/alerting/components/notifications/utils'
import {getResourceList} from 'src/alerting/selectors'
import {useRuleDispatch} from './RuleOverlayProvider'

// Types
import {
  NotificationEndpoint,
  NotificationRuleDraft,
  AppState,
  ResourceType,
} from 'src/types'

interface StateProps {
  endpoints: NotificationEndpoint[]
}

interface OwnProps {
  rule: NotificationRuleDraft
}

type Props = OwnProps & StateProps

const RuleMessage: FC<Props> = ({endpoints, rule}) => {
  const dispatch = useRuleDispatch()

  const onSelectEndpoint = endpointID => {
    dispatch({
      type: 'UPDATE_RULE',
      rule: {
        ...rule,
        ...getRuleVariantDefaults(endpoints, endpointID),
        endpointID,
      },
    })
  }

  useEffect(() => {
    if (!rule.endpointID && endpoints.length) {
      onSelectEndpoint(endpoints[0].id)
    }
  }, [])

  return (
    <Grid.Row>
      <Grid.Column widthSM={Columns.Two}>Message</Grid.Column>
      <Grid.Column widthSM={Columns.Ten}>
        <Panel>
          <Panel.Body>
            <Form.Element label="Notification Endpoint">
              <RuleEndpointDropdown
                endpoints={endpoints}
                onSelectEndpoint={onSelectEndpoint}
                selectedEndpointID={rule.endpointID}
              />
            </Form.Element>
            <RuleMessageContents rule={rule} />
          </Panel.Body>
        </Panel>
      </Grid.Column>
    </Grid.Row>
  )
}

const mstp = (state: AppState) => {
  return {
    endpoints: getResourceList<NotificationEndpoint>(
      state,
      ResourceType.NotificationEndpoints
    ),
  }
}

export default connect(mstp)(RuleMessage)
