import "./Message.scss"
import { fetchMusic } from "../../api/api"
import { User, MessageUpdate, MessageType, Message, Messages } from "../../model/snowcast_pb"
interface MessageProps {
      Message: Message
}

export function TextMessage(props: MessageProps) {
      return <div className="text-message">
            <div className="sender">{props.Message.getSender()}</div>
            <div className="bodytext">{props.Message.getMessage()}</div>
      </div>
}

export function MusicMessage(props: MessageProps) {
      const sender = props.Message.getSender()
      const musicName = props.Message.getMessage()

      async function playMusic(music: string) {
            fetchMusic(music).then((audioNode: AudioBufferSourceNode) => audioNode.start())
      }

      return <div className="music-message">
            <div className="sender">{props.Message.getSender()}</div>
            <div className="bodytext">
                  <div className="play-icon">
                        <svg onClick={() => { playMusic(musicName) }} version="1.1" id="play" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" x="0px" y="0px" height="90%" width="90%" viewBox="0 0 100 100" enable-background="new 0 0 100 100" xmlSpace="preserve">
                              <path className="stroke-solid" fill="none" stroke="#fc0339" d="M49.9,2.5C23.6,2.8,2.1,24.4,2.5,50.4C2.9,76.5,24.7,98,50.3,97.5c26.4-0.6,47.4-21.8,47.2-47.7C97.3,23.7,75.7,2.3,49.9,2.5" />
                              <path className="icon" fill="#fc0339" d="M38,69c-1,0.5-1.8,0-1.8-1.1V32.1c0-1.1,0.8-1.6,1.8-1.1l34,18c1,0.5,1,1.4,0,1.9L38,69z" />
                        </svg>
                  </div>
                  {props.Message.getMessage()}
            </div>

      </div>
}