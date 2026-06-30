import { useCall } from '../contexts/CallContext';
import CallOverlay from './CallOverlay';
import IncomingCallModal from './IncomingCallModal';

export default function CallUI() {
  const call = useCall();

  return (
    <>
      {call.isRinging && <IncomingCallModal />}
      <CallOverlay />
    </>
  );
}
