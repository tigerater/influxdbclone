// Libraries
import React, {FC, ReactChild, useState} from 'react'

// Components
import {
  Panel,
  InfluxColors,
  DapperScrollbars,
  Input,
  IconFont,
  FlexBox,
  FlexDirection,
  ComponentSize,
} from '@influxdata/clockface'
import QuestionMarkTooltip from 'src/shared/components/question_mark_tooltip/QuestionMarkTooltip'

interface Props {
  title: string
  createButton: JSX.Element
  questionMarkTooltipContents: ReactChild
  children: (searchTerm: string) => ReactChild
}

const AlertsColumnHeader: FC<Props> = ({
  children,
  title,
  createButton,
  questionMarkTooltipContents,
}) => {
  const [searchTerm, onChangeSearchTerm] = useState('')
  return (
    <Panel
      backgroundColor={InfluxColors.Kevlar}
      className="alerting-index--column"
    >
      <Panel.Header>
        <FlexBox direction={FlexDirection.Row} margin={ComponentSize.Small}>
          <Panel.Title style={{fontSize: '17px', width: 'auto'}}>
            {title}
          </Panel.Title>
          <QuestionMarkTooltip
            testID={`${title}--question-mark`}
            tipContent={questionMarkTooltipContents}
          />
        </FlexBox>
        {createButton}
      </Panel.Header>
      <div className="alerting-index--search">
        <Input
          icon={IconFont.Search}
          placeholder={`Filter ${title}...`}
          value={searchTerm}
          onChange={e => onChangeSearchTerm(e.target.value)}
        />
      </div>
      <div className="alerting-index--column-body">
        <DapperScrollbars
          autoSize={false}
          autoHide={true}
          style={{width: '100%', height: '100%'}}
        >
          <div className="alerting-index--list">{children(searchTerm)}</div>
        </DapperScrollbars>
      </div>
    </Panel>
  )
}

export default AlertsColumnHeader
