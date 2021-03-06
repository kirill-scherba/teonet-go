#ifndef EVENT_H
#define EVENT_H

/**
 * KSNet event manager events
 */
typedef enum ksnetEvMgrEvents {

  /**
   * #0 Calls immediately after event manager starts
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data NULL
   * @param data_len 0
   * @param user_data NULL
   */
  EV_K_STARTED, // #0  Calls immediately after event manager starts

  /**
   * #1 Calls before event manager stopped
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data NULL
   * @param data_len 0
   * @param user_data NULL
   */
  EV_K_STOPPED_BEFORE, // #1  Calls before event manager stopped

  /**
   * #2  Calls after event manager stopped
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data NULL
   * @param data_len 0
   * @param user_data NULL
   */
  EV_K_STOPPED, // #2  Calls after event manager stopped

  /**
   * #3 New peer connected to host event
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to ksnCorePacketData
   * @param data_len Size of ksnCorePacketData
   * @param user_data NULL
   */
  EV_K_CONNECTED, // #3  New peer connected to host

  /**
   * #4  A peer was disconnected from host
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to ksnCorePacketData
   * @param data_len Size of ksnCorePacketData
   * @param user_data NULL
   */
  EV_K_DISCONNECTED, // #4  A peer was disconnected from host

  /**
   * #5  This host Received a data
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to ksnCorePacketData
   * @param data_len Size of ksnCorePacketData
   * @param user_data NULL
   *
   */
  EV_K_RECEIVED,       // #5  This host Received a data
  EV_K_RECEIVED_WRONG, ///< #6  Wrong packet received
  /**
   * #7 This host Received ACK to sent data
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to ksnCorePacketData
   * @param data_len Size of ksnCorePacketData
   * @param user_data Pointer to packet ID
   */
  EV_K_RECEIVED_ACK, ///< #7  This host Received ACK to sent data
  EV_K_IDLE, ///< #8  Idle check host events (after 11.5 after last host send or
             ///< receive data)
  EV_K_TIMER, ///< #9  Timer event

  /**
   * #10 Hotkey event
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to integer hotkey
   * @param data_len Size of integer
   * @param user_data Pointer to raw keyboard input buffer
   */
  EV_K_HOTKEY, ///< #10 Hotkey event
  EV_K_USER,   ///< #11 User press A hotkey
  EV_K_ASYNC,  ///< #12 Async event

  /**
   * #13 After terminal started (in place to define commands
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data NULL
   * @param data_len 0
   * @param user_data NULL
   */
  EV_K_TERM_STARTED, // #13 After terminal started (in place to define commands

  /**
   * #14 Teonet Callback QUEUE event
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data Pointer to ksnCQueData
   * @param data_len Size of ksnCQueData structure
   * @param user_data Pointer to integer with type of this event:
   *                  1 - success; 0 - timeout
   */
  EV_K_CQUE_CALLBACK, // #14 Teonet Callback QUEUE event

  EV_K_STREAM_CONNECTED,       ///< #15 After stream connected
  EV_K_STREAM_CONNECT_TIMEOUT, ///< #16 Connection timeout
  EV_K_STREAM_DISCONNECTED,    ///< #17 After stream disconnected
  EV_K_STREAM_DATA,            ///< #18 Input stream has a data

  EV_K_SUBSCRIBE,  ///< #19 Subscribe answer command received
  EV_K_SUBSCRIBED, ///< #20 A peer subscribed to event at this host

  EV_K_L0_CONNECTED,    ///< #21 New L0 client connected to L0 server
  EV_K_L0_DISCONNECTED, ///< #22 A L0 client was disconnected from L0 server
  EV_K_L0_NEW_VISIT, ///< #23 New clients visit event to all subscribers (equal
                     ///< to L0_CONNECTED but send number of visits)

  EV_H_HWS_EVENT, ///< #24 HTTP WebSocket server event, data - depends of event,
                  ///< user data - poiter to hws_event

  EV_U_RECEIVED, ///< #25 UNIX socket received data event, data - data received
                 ///< from unix socket, user_data - pointer to the usock_class

  EV_D_SET, ///< #26 Database updated

  /**
   * #27 Angular interval event happened
   *
   * This event sends by Angular Teonet Service running in Teonet Node
   * Application when Angular interval tick happened
   *
   * Parameters of Teonet Events callback function:
   *
   * @param ke Pointer to ksnetEvMgrClass
   * @param event This event
   * @param data NULL
   * @param data_len 0
   * @param user_data NULL
   */
  EV_A_INTERVAL, // #27 Angular interval event happened

  EV_K_LOGGING, ///< #28 Logging server event, like EV_K_RECEIVED: data Pointer
                ///< to ksnCorePacketData, data_len Size of ksnCorePacketData, ,
                ///< user_data NULL

  EV_K_LOG_READER, ///< #29 LogReader read data.

  EV_K_APP_USER = 0x8000 ///< #0x8000 Teonet based Applications events

} ksnetEvMgrEvents;

#endif /* EVENT_H */
