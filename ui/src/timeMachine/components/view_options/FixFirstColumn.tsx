// Libraries
import React, {SFC} from 'react'

// Components
import {Form, SlideToggle, FlexBox, Grid} from '@influxdata/clockface'

// Types
import {Columns, FlexDirection, ComponentSize} from '@influxdata/clockface'

interface Props {
  fixed: boolean
  onToggleFixFirstColumn: () => void
}

const GraphOptionsFixFirstColumn: SFC<Props> = ({
  fixed,
  onToggleFixFirstColumn,
}) => (
  <Grid.Column widthXS={Columns.Twelve}>
    <Form.Element label="First Column">
      <Form.Box>
        <FlexBox direction={FlexDirection.Row} margin={ComponentSize.Small}>
          <SlideToggle.Label text="Scroll with table" />
          <SlideToggle
            active={fixed}
            onChange={onToggleFixFirstColumn}
            size={ComponentSize.ExtraSmall}
          />
          <SlideToggle.Label text="Fixed" />
        </FlexBox>
      </Form.Box>
    </Form.Element>
  </Grid.Column>
)

export default GraphOptionsFixFirstColumn
