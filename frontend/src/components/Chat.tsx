import React, { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useChat } from '../../context/ChatContext';
import { useAuth } from '../../context/AuthContext';
import { MessageList } from './MessageList';
import { MessageInput } from './MessageInput';
import { UserList } from './UserList';
import { MatchInfo } from './MatchInfo';
import { EventList } from './EventList';
import { LoadingSpinner } from '../shared/LoadingSpinner';

export const ChatRoom: React.FC = () => {
  const { matchId } = useParams<{ matchId: string }>();
  const { messages, events, sendMessage, joinRoom, leaveRoom, isConnected } = useChat();
  const { user } = useAuth();
  const [isLoading, setIsLoading] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (matchId && user) {
      joinRoom(matchId);
      setIsLoading(false);
    }

    return () => {
      leaveRoom();
    };
  }, [matchId, user]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  if (isLoading) {
    return <LoadingSpinner />;
  }

  return (
    <div className="flex h-screen bg-gray-100">
      <div className="flex flex-row w-full max-w-7xl mx-auto bg-white shadow-xl">
        {/* Left Sidebar - Match Info & Events */}
        <div className="w-64 flex flex-col border-r bg-gray-50">
          <MatchInfo />
          <div className="flex-1 overflow-y-auto">
            <EventList events={events} />
          </div>
        </div>

        {/* Main Chat Area */}
        <div className="flex flex-col flex-1">
          {/* Connection Status */}
          <div className="px-4 py-2 bg-gray-800 text-white">
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${
                isConnected ? 'bg-green-500' : 'bg-red-500'
              }`} />
              <span className="text-sm">
                {isConnected ? 'Connected' : 'Connecting...'}
              </span>
            </div>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-4 py-6">
            <MessageList messages={messages} currentUser={user} />
            <div ref={bottomRef} />
          </div>

          {/* Message Input */}
          <div className="p-4 border-t">
            <MessageInput onSend={content => sendMessage(content, matchId!)} />
          </div>
        </div>

        {/* Right Sidebar - Online Users */}
        <div className="w-64 border-l bg-gray-50">
          <UserList />
        </div>
      </div>
    </div>
  );
};