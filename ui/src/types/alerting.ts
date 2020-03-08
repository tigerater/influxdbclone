import {
  StatusRule,
  NotificationRuleBase,
  TagRule,
  SlackNotificationRuleBase,
  SMTPNotificationRuleBase,
  PagerDutyNotificationRuleBase,
  NotificationRule,
} from 'src/client'

type Omit<T, U> = Pick<T, Exclude<keyof T, U>>
type Overwrite<T, U> = Omit<T, keyof U> & U

interface WithClientID<T> {
  cid: string
  value: T
}

export type StatusRuleDraft = WithClientID<StatusRule>

export type TagRuleDraft = WithClientID<TagRule>

// TODO: Spec this out in the OpenAPI spec instead. It should be whatever the
// server accepts as the request body for a `POST /api/v2/notificationRules`
export type NewNotificationRule = Omit<NotificationRule, 'id'>

export type NotificationRuleBaseDraft = Overwrite<
  NotificationRuleBase,
  {
    id?: string
    statusRules: StatusRuleDraft[]
    tagRules: TagRuleDraft[]
  }
>

export type NotificationRuleDraft = SlackRule | SMTPRule | PagerDutyRule

type SlackRule = NotificationRuleBaseDraft & SlackNotificationRuleBase
type SMTPRule = NotificationRuleBaseDraft & SMTPNotificationRuleBase
type PagerDutyRule = NotificationRuleBaseDraft & PagerDutyNotificationRuleBase

export {
  Check,
  Threshold,
  CheckBase,
  StatusRule,
  LevelRule,
  TagRule,
  CheckStatusLevel,
  GreaterThreshold,
  LesserThreshold,
  RangeThreshold,
  ThresholdCheck,
  DeadmanCheck,
  NotificationEndpoint,
  NotificationRuleBase,
  NotificationRule,
  SMTPNotificationRuleBase,
  SlackNotificationRuleBase,
  PagerDutyNotificationRuleBase,
  SMTPNotificationRule,
  SlackNotificationRule,
  PagerDutyNotificationRule,
} from '../client'

import {Check, Threshold} from '../client'

export type CheckType = Check['type']
export type ThresholdType = Threshold['type']

export type AlertHistoryType = 'statuses' | 'notifications'
