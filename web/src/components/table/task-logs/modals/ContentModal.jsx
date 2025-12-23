/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect } from 'react';
import { Modal, Button, Typography, Spin } from '@douyinfe/semi-ui';
import { IconExternalOpen, IconCopy, IconDownload } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const ContentModal = ({
  isModalOpen,
  setIsModalOpen,
  modalContent,
  isVideo,
  isImage,
}) => {
  const { t } = useTranslation();
  const [videoError, setVideoError] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isModalOpen && isVideo) {
      setVideoError(false);
      setIsLoading(true);
    }
    if (isModalOpen && isImage) {
      setImageError(false);
      setIsLoading(true);
    }
  }, [isModalOpen, isVideo, isImage]);

  const handleVideoError = () => {
    setVideoError(true);
    setIsLoading(false);
  };

  const handleVideoLoaded = () => {
    setIsLoading(false);
  };

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(modalContent);
  };

  const handleOpenInNewTab = () => {
    window.open(modalContent, '_blank');
  };

  const handleDownload = () => {
    // 创建一个隐藏的 a 标签来下载文件
    const link = document.createElement('a');
    link.href = modalContent;
    // 从 URL 中提取文件名
    const urlParts = modalContent.split('/');
    const fileName =
      urlParts[urlParts.length - 1] || (isVideo ? 'video.mp4' : 'image.png');
    link.download = fileName;
    link.target = '_blank';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const renderVideoContent = () => {
    if (videoError) {
      return (
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '16px' }}
          >
            {t('视频无法在当前浏览器中播放，这可能是由于：')}
          </Text>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '8px', fontSize: '12px' }}
          >
            {t('• 视频服务商的跨域限制')}
          </Text>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '8px', fontSize: '12px' }}
          >
            {t('• 需要特定的请求头或认证')}
          </Text>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '16px', fontSize: '12px' }}
          >
            {t('• 防盗链保护机制')}
          </Text>

          <div style={{ marginTop: '20px' }}>
            <Button
              icon={<IconExternalOpen />}
              onClick={handleOpenInNewTab}
              style={{ marginRight: '8px' }}
            >
              {t('在新标签页中打开')}
            </Button>
            <Button icon={<IconCopy />} onClick={handleCopyUrl}>
              {t('复制链接')}
            </Button>
          </div>

          <div
            style={{
              marginTop: '16px',
              padding: '8px',
              backgroundColor: '#f8f9fa',
              borderRadius: '4px',
            }}
          >
            <Text
              type='tertiary'
              style={{ fontSize: '10px', wordBreak: 'break-all' }}
            >
              {modalContent}
            </Text>
          </div>
        </div>
      );
    }

    return (
      <div>
        <div style={{ position: 'relative' }}>
          {isLoading && (
            <div
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                zIndex: 10,
              }}
            >
              <Spin size='large' />
            </div>
          )}
          <video
            src={modalContent}
            controls
            style={{ width: '100%', maxHeight: '60vh' }}
            autoPlay
            crossOrigin='anonymous'
            onError={handleVideoError}
            onLoadedData={handleVideoLoaded}
            onLoadStart={() => setIsLoading(true)}
          />
        </div>
        <div style={{ marginTop: '16px', textAlign: 'center' }}>
          <Button
            icon={<IconDownload />}
            onClick={handleDownload}
            style={{ marginRight: '8px' }}
          >
            {t('下载')}
          </Button>
          <Button icon={<IconCopy />} onClick={handleCopyUrl}>
            {t('复制链接')}
          </Button>
        </div>
      </div>
    );
  };

  const handleImageError = () => {
    setImageError(true);
    setIsLoading(false);
  };

  const handleImageLoaded = () => {
    setIsLoading(false);
  };

  const renderImageContent = () => {
    if (imageError) {
      return (
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '16px' }}
          >
            {t('图片无法加载，这可能是由于：')}
          </Text>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '8px', fontSize: '12px' }}
          >
            {t('• 图片服务商的跨域限制')}
          </Text>
          <Text
            type='tertiary'
            style={{ display: 'block', marginBottom: '16px', fontSize: '12px' }}
          >
            {t('• 图片链接已过期')}
          </Text>

          <div style={{ marginTop: '20px' }}>
            <Button
              icon={<IconExternalOpen />}
              onClick={handleOpenInNewTab}
              style={{ marginRight: '8px' }}
            >
              {t('在新标签页中打开')}
            </Button>
            <Button icon={<IconCopy />} onClick={handleCopyUrl}>
              {t('复制链接')}
            </Button>
          </div>

          <div
            style={{
              marginTop: '16px',
              padding: '8px',
              backgroundColor: '#f8f9fa',
              borderRadius: '4px',
            }}
          >
            <Text
              type='tertiary'
              style={{ fontSize: '10px', wordBreak: 'break-all' }}
            >
              {modalContent}
            </Text>
          </div>
        </div>
      );
    }

    return (
      <div>
        <div style={{ position: 'relative', textAlign: 'center' }}>
          {isLoading && (
            <div
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                zIndex: 10,
              }}
            >
              <Spin size='large' />
            </div>
          )}
          <img
            src={modalContent}
            alt='Preview'
            style={{
              maxWidth: '100%',
              maxHeight: '60vh',
              objectFit: 'contain',
            }}
            onError={handleImageError}
            onLoad={handleImageLoaded}
          />
        </div>
        <div style={{ marginTop: '16px', textAlign: 'center' }}>
          <Button
            icon={<IconDownload />}
            onClick={handleDownload}
            style={{ marginRight: '8px' }}
          >
            {t('下载')}
          </Button>
          <Button icon={<IconCopy />} onClick={handleCopyUrl}>
            {t('复制链接')}
          </Button>
        </div>
      </div>
    );
  };

  const renderContent = () => {
    if (isVideo) {
      return renderVideoContent();
    }
    if (isImage) {
      return renderImageContent();
    }
    return <p style={{ whiteSpace: 'pre-line' }}>{modalContent}</p>;
  };

  return (
    <Modal
      visible={isModalOpen}
      onOk={() => setIsModalOpen(false)}
      onCancel={() => setIsModalOpen(false)}
      closable={null}
      bodyStyle={{
        overflow: 'hidden',
        padding:
          (isVideo && videoError) || (isImage && imageError) ? '0' : '24px',
      }}
      width={800}
    >
      {renderContent()}
    </Modal>
  );
};

export default ContentModal;
