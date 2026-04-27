import React from 'react';

interface FooterProps {
  copyright?: string;
  links?: Array<{ label: string; href: string }>;
}

const Footer: React.FC<FooterProps> = ({
  copyright = `© ${new Date().getFullYear()} Interview Prep Platform`,
  links = [
    { label: 'Privacy Policy', href: '/privacy' },
    { label: 'Terms of Service', href: '/terms' },
    { label: 'Contact', href: '/contact' },
  ],
}) => {
  return (
    <footer className="footer">
      <p className="footer__copyright">{copyright}</p>
      <nav className="footer__nav">
        {links.map((link) => (
          <a key={link.href} href={link.href} className="footer__link">
            {link.label}
          </a>
        ))}
      </nav>
    </footer>
  );
};

export default Footer;
