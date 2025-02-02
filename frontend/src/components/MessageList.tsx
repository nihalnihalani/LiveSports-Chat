import React from 'react';
import { format } from 'date-fns';
import { Message } from '../../types';

interface Props {
  messages: Message[];
  currentUser: any;
}

export const MessageList: React.FC<Props> = ({ messages, currentUser }) => {
  return (
    <div className="space-y-4">
      {messages.map((message) => (
        <div
          key={message.id}
          className={`flex flex-col ${
            message.userId === currentUser?.id ? 'items-end' : 'items-start'
          }`}
        >
          <div className="flex items-center space-x-2 mb-1">
            <span className="text-sm font-medium text-gray-700">
              {message.username}
            </span>
            <span className="text-xs bg-gray-200 px-2 py-0.5 rounded-full">
              {message.teamName}
            </span>
            <span className="text-xs text-gray-500">
              {format(new Date(message.timestamp), 'HH:mm')}
            </span>
          </div>
          <div
            className={`px-4 py-2 rounded-lg max-w-xs lg:max-w-md ${
              message.userId === currentUser?.id
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-900'
            }`}
          >
            {message.type === 'event' && (
              <span className="text-yellow-500 mr-2">🏆</span>
            )}
            {message.content}
          </div>
        </div>
      ))}
    </div>
  );
};