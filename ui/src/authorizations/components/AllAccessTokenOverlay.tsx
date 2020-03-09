import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'

// Components
import {
  Alert,
  IconFont,
  ComponentColor,
  FlexBox,
  AlignItems,
  FlexDirection,
  ComponentSize,
  Button,
  ButtonType,
  Input,
  Overlay,
  Form,
} from '@influxdata/clockface'

// Actions
import {createAuthorization} from 'src/authorizations/actions/thunks'

// Utils
import {allAccessPermissions} from 'src/authorizations/utils/permissions'
import {getOrg} from 'src/organizations/selectors'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

// Types
import {AppState, Authorization} from 'src/types'

interface OwnProps {
  onClose: () => void
}

interface StateProps {
  orgID: string
}

interface DispatchProps {
  onCreateAuthorization: typeof createAuthorization
}

interface State {
  description: string
}

type Props = OwnProps & StateProps & DispatchProps

@ErrorHandling
class AllAccessTokenOverlay extends PureComponent<Props, State> {
  public state = {description: ''}

  render() {
    const {description} = this.state

    return (
      <Overlay.Container maxWidth={500}>
        <Overlay.Header
          title="Generate All Access Token"
          onDismiss={this.handleDismiss}
        />
        <Overlay.Body>
          <Form onSubmit={this.handleSave}>
            <FlexBox
              alignItems={AlignItems.Center}
              direction={FlexDirection.Column}
              margin={ComponentSize.Large}
            >
              <Alert
                icon={IconFont.AlertTriangle}
                color={ComponentColor.Warning}
              >
                This token will be able to create, update, delete, read, and
                write to anything in this organization
              </Alert>
              <Form.Element label="Description">
                <Input
                  placeholder="Describe this new token"
                  value={description}
                  onChange={this.handleInputChange}
                />
              </Form.Element>

              <Form.Footer>
                <Button
                  text="Cancel"
                  icon={IconFont.Remove}
                  onClick={this.handleDismiss}
                />

                <Button
                  text="Save"
                  icon={IconFont.Checkmark}
                  color={ComponentColor.Success}
                  type={ButtonType.Submit}
                />
              </Form.Footer>
            </FlexBox>
          </Form>
        </Overlay.Body>
      </Overlay.Container>
    )
  }

  private handleSave = () => {
    const {orgID, onCreateAuthorization} = this.props

    const token: Authorization = {
      orgID,
      description: this.state.description,
      permissions: allAccessPermissions(orgID),
    }

    onCreateAuthorization(token)

    this.handleDismiss()
  }

  private handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const {value} = e.target

    this.setState({description: value})
  }

  private handleDismiss = () => {
    this.props.onClose()
  }
}

const mstp = (state: AppState): StateProps => {
  return {
    orgID: getOrg(state).id,
  }
}

const mdtp: DispatchProps = {
  onCreateAuthorization: createAuthorization,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(AllAccessTokenOverlay)
