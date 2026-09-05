-- Personal introduction on referral qualification applications.
-- Commission slots are limited; reviewers need applicant context.

ALTER TABLE referral_code
  ADD COLUMN personal_intro VARCHAR(2000) NOT NULL DEFAULT ''
  COMMENT 'applicant self-introduction for limited-slot review';
