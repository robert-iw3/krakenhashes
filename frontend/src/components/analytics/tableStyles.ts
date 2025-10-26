/**
 * Shared table styling constants for analytics sections
 * Ensures consistent column widths and alignments across all analytics tables
 */

// Standard 3-column table (Label | Count | Percentage)
export const threeColumnTableStyles = {
  labelCell: {
    width: '60%',
  },
  countCell: {
    width: '20%',
    textAlign: 'right' as const,
  },
  percentageCell: {
    width: '20%',
    textAlign: 'right' as const,
  },
};

// Standard 2-column table (Metric | Value)
export const twoColumnTableStyles = {
  labelCell: {
    width: '70%',
  },
  valueCell: {
    width: '30%',
    textAlign: 'right' as const,
  },
};

// Password Reuse table (5 columns)
export const passwordReuseTableStyles = {
  passwordCell: {
    width: '15%',
  },
  usersCell: {
    width: '50%',
  },
  occurrencesCell: {
    width: '15%',
    textAlign: 'right' as const,
  },
  userCountCell: {
    width: '12%',
    textAlign: 'right' as const,
  },
  actionsCell: {
    width: '8%',
    textAlign: 'center' as const,
  },
};

// Top Passwords table (3 columns with different layout)
export const topPasswordsTableStyles = {
  passwordCell: {
    width: '50%',
  },
  countCell: {
    width: '25%',
    textAlign: 'right' as const,
  },
  percentageCell: {
    width: '25%',
    textAlign: 'right' as const,
  },
};

// Mask Analysis table (4 columns)
export const maskAnalysisTableStyles = {
  maskCell: {
    width: '35%',
  },
  exampleCell: {
    width: '30%',
  },
  countCell: {
    width: '17.5%',
    textAlign: 'right' as const,
  },
  percentageCell: {
    width: '17.5%',
    textAlign: 'right' as const,
  },
};
