import React from 'react';

import {t} from './i18n';

// React unmounts the whole tree when a render throws, which turned one bad screen into a
// blank page. This keeps the failure inside the screen that caused it: the header, the tabs
// and the chat stay usable, and the player can walk out of the broken tab on their own.
export default class Boundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {failed: false};
  }

  static getDerivedStateFromError() {
    return {failed: true};
  }

  componentDidCatch(error) {
    // Left in the console on purpose: a blank screen with no trace is what made the original
    // report impossible to act on.
    console.error('vpet: screen failed', error);
  }

  componentDidUpdate(prev) {
    // Moving to another tab is the way out, so a new screen starts clean.
    if (this.state.failed && prev.resetKey !== this.props.resetKey) {
      this.setState({failed: false});
    }
  }

  render() {
    if (!this.state.failed) {
      return this.props.children;
    }
    return (
      <div className='vpet-broken' role='alert'>
        <p>{t('app.screenFailed')}</p>
        <button type='button' className='vpet-btn vpet-btn--primary' onClick={this.props.onHome}>{t('app.backToTown')}</button>
      </div>
    );
  }
}
