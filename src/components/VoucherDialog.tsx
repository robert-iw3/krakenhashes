import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  FormControlLabel,
  Switch,
  Typography,
  Box
} from '@mui/material';
import { useVouchers } from '../hooks/useVouchers';

interface VoucherDialogProps {
  open: boolean;
  onClose: () => void;
}

export const VoucherDialog: React.FC<VoucherDialogProps> = ({ open, onClose }) => {
  const [isContinuous, setIsContinuous] = React.useState(false);
  const { tempVoucher, createTempVoucher, confirmVoucher } = useVouchers();

  React.useEffect(() => {
    if (open && !tempVoucher) {
      createTempVoucher();
    }
  }, [open, tempVoucher]);

  const handleConfirm = async () => {
    if (tempVoucher) {
      await confirmVoucher(tempVoucher.code, isContinuous);
      onClose();
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Create New Agent Voucher</DialogTitle>
      <DialogContent>
        <Box sx={{ my: 2 }}>
          <FormControlLabel
            control={
              <Switch
                checked={isContinuous}
                onChange={(e) => setIsContinuous(e.target.checked)}
              />
            }
            label="Allow Multiple Uses"
          />
          <Typography variant="caption" display="block" sx={{ mt: 1 }}>
            Multiple-use vouchers can be disabled later through Agent Management
          </Typography>
        </Box>
        {tempVoucher && (
          <Typography variant="h5" sx={{ mt: 2, textAlign: 'center' }}>
            {tempVoucher.code}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleConfirm} variant="contained" color="primary">
          Create Voucher
        </Button>
      </DialogActions>
    </Dialog>
  );
}; 