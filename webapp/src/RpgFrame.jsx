import React from 'react';

export default function RpgFrame({as: Tag = 'section', className = '', children, ...props}) {
  return <Tag className={`vpet-rpg-frame ${className}`.trim()} {...props}>{children}</Tag>;
}
