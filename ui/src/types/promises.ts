export interface CancelBox<T> {
  promise: Promise<T>
  cancel: () => void
}

export class CancellationError extends Error {
  constructor() {
    super(...arguments)

    this.name = 'CancellationError'
  }
}
