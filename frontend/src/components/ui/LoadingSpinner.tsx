import React from 'react';

const LoadingSpinner: React.FC = () => {
  return (
    <div className="flex justify-center py-12">
      <div className="w-8 h-8 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin"></div>
    </div>
  );
};

export default LoadingSpinner;
