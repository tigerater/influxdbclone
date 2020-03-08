// Libraries
import React, {FC} from 'react'

// Components
import SlackMessage from './SlackMessage'
import SMTPMessage from './SMTPMessage'
import PagerDutyMessage from './PagerDutyMessage'

// Types
import {NotificationRuleDraft} from 'src/types'

// Hooks
import {useRuleDispatch} from 'src/shared/hooks'

interface Props {
  rule: NotificationRuleDraft
}

const RuleMessageContents: FC<Props> = ({rule}) => {
  const dispatch = useRuleDispatch()
  const onChange = ({target}) => {
    const {name, value} = target

    dispatch({
      type: 'UPDATE_RULE',
      rule: {
        ...rule,
        [name]: value,
      },
    })
  }

  switch (rule.type) {
    case 'slack': {
      const {messageTemplate, channel} = rule
      return (
        <SlackMessage
          messageTemplate={messageTemplate}
          channel={channel}
          onChange={onChange}
        />
      )
    }
    case 'smtp': {
      const {to, subjectTemplate, bodyTemplate} = rule
      return (
        <SMTPMessage
          to={to}
          onChange={onChange}
          bodyTemplate={bodyTemplate}
          subjectTemplate={subjectTemplate}
        />
      )
    }
    case 'pagerduty': {
      const {messageTemplate} = rule
      return (
        <PagerDutyMessage
          messageTemplate={messageTemplate}
          onChange={onChange}
        />
      )
    }

    default:
      throw new Error('Unexpected endpoint type in <RuleMessageContents/>.')
  }
}

export default RuleMessageContents
