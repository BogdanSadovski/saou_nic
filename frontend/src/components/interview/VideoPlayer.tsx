import React, { useRef, useState } from 'react';

interface VideoPlayerProps {
  src?: string;
  poster?: string;
  autoPlay?: boolean;
  onEnded?: () => void;
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({
  src,
  poster,
  autoPlay = false,
  onEnded,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [isPlaying, setIsPlaying] = useState(autoPlay);

  const togglePlay = () => {
    const video = videoRef.current;
    if (!video) return;
    if (isPlaying) {
      video.pause();
    } else {
      video.play();
    }
    setIsPlaying(!isPlaying);
  };

  return (
    <div className="video-player">
      <video
        ref={videoRef}
        className="video-player__video"
        src={src}
        poster={poster}
        autoPlay={autoPlay}
        onEnded={() => {
          setIsPlaying(false);
          onEnded?.();
        }}
        controls
      >
        Your browser does not support the video tag.
      </video>
      <button className="video-player__toggle" onClick={togglePlay}>
        {isPlaying ? 'Pause' : 'Play'}
      </button>
    </div>
  );
};

export default VideoPlayer;
