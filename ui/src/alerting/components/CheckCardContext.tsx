// Libraries
import React, {FunctionComponent} from 'react'

// Components
import {Context, IconFont} from 'src/clockface'
import {ComponentColor} from '@influxdata/clockface'

interface Props {
  onDelete: () => void
  onClone: () => void
  onExport: () => void
}

const CheckCardContext: FunctionComponent<Props> = ({
  onDelete,
  onClone,
  onExport,
}) => {
  return (
    <Context>
      <Context.Menu icon={IconFont.CogThick}>
        <Context.Item label="Export" action={onExport} />
      </Context.Menu>
      <Context.Menu icon={IconFont.Duplicate} color={ComponentColor.Secondary}>
        <Context.Item label="Clone" action={onClone} />
      </Context.Menu>
      <Context.Menu
        icon={IconFont.Trash}
        color={ComponentColor.Danger}
        testID="context-delete-menu"
      >
        <Context.Item
          label="Delete"
          action={onDelete}
          testID="context-delete-task"
        />
      </Context.Menu>
    </Context>
  )
}

export default CheckCardContext
